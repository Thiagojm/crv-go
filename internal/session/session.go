package session

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Thiagojm/crv-go/internal/store"
)

const (
	leaseTTL    = 15 * time.Second
	maxTimingMs = 5000
	positions   = "ABCD"
)

var (
	ErrCatalogNotReady = errors.New("catálogo não está pronto")
	ErrTooFewImages    = errors.New("são necessárias pelo menos 4 imagens elegíveis")
	ErrActiveExists    = errors.New("já existe uma sessão em andamento")
	ErrNotFound        = errors.New("sessão não encontrada")
	ErrConflict        = errors.New("conflito de revisão")
	ErrBadState        = errors.New("estado inválido para esta operação")
	ErrLease           = errors.New("esta aba não possui a edição")
	ErrChoiceConflict  = errors.New("escolha conflitante")
	ErrRand            = errors.New("falha ao obter aleatoriedade")
	ErrValidation      = errors.New("registro inválido")
	ErrPaused          = errors.New("sessão pausada")
)

type Service struct {
	Store *store.Store
	Now   func() time.Time
	Rand  io.Reader
}

func New(st *store.Store) *Service {
	return &Service{
		Store: st,
		Now:   func() time.Time { return time.Now().UTC() },
		Rand:  rand.Reader,
	}
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) reader() io.Reader {
	if s.Rand != nil {
		return s.Rand
	}
	return rand.Reader
}

type CreateInput struct {
	OperationID   string
	Disposition   string
	Concentration string
}

type ChoiceInput struct {
	ExpectedRevision int
	Choice           string
	Confidence       *int
	LeaseToken       string
}

func (s *Service) Create(in CreateInput) (*store.SessionRow, string, error) {
	if in.OperationID == "" {
		id, err := randomHex(s.reader(), 16)
		if err != nil {
			return nil, "", ErrRand
		}
		in.OperationID = id
	}
	if existing, err := s.Store.SessionByOpID(in.OperationID); err != nil {
		return nil, "", err
	} else if existing != nil {
		return existing, existing.LeaseToken, nil
	}
	if active, err := s.Store.ActiveSession(); err != nil {
		return nil, "", err
	} else if active != nil {
		return nil, "", ErrActiveExists
	}
	cat, err := s.Store.ActiveCatalog()
	if err != nil {
		return nil, "", err
	}
	if cat == nil || cat.EligibleCount < 4 {
		return nil, "", ErrCatalogNotReady
	}
	hashes, err := s.Store.EligibleSHAs(cat.RevisionID)
	if err != nil {
		return nil, "", err
	}
	if len(hashes) < 4 {
		return nil, "", ErrTooFewImages
	}
	target, order, err := drawAssignment(s.reader(), hashes)
	if err != nil {
		return nil, "", err
	}
	code, err := randomCode(s.reader())
	if err != nil {
		return nil, "", err
	}
	id, err := randomHex(s.reader(), 16)
	if err != nil {
		return nil, "", err
	}
	lease, err := randomHex(s.reader(), 16)
	if err != nil {
		return nil, "", err
	}
	rec := EmptyRecord()
	rec.Disposition = in.Disposition
	rec.Concentration = in.Concentration
	if err := ValidateRecord(&rec); err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrValidation, err)
	}
	recJSON, err := json.Marshal(rec)
	if err != nil {
		return nil, "", err
	}
	proto, err := json.Marshal(SnapshotProtocol())
	if err != nil {
		return nil, "", err
	}
	ts := s.now().Format(time.RFC3339)
	row := &store.SessionRow{
		ID:                 id,
		Code:               code,
		CreateOpID:         in.OperationID,
		State:              "collecting",
		Revision:           1,
		CatalogRevisionID:  cat.RevisionID,
		TargetSHA:          target,
		PosA:               order[0],
		PosB:               order[1],
		PosC:               order[2],
		PosD:               order[3],
		ProtocolJSON:       string(proto),
		HelpVersion:        HelpVersion,
		RecordJSON:         string(recJSON),
		Step:               1,
		CreatedAt:          ts,
		UpdatedAt:          ts,
		LeaseToken:         lease,
		LeaseUntil:         s.now().Add(leaseTTL).Format(time.RFC3339),
		OpenedExamplesJSON: "[]",
	}
	if err := s.Store.InsertSession(row); err != nil {
		if store.IsUniqueErr(err) {
			if again, e2 := s.Store.SessionByOpID(in.OperationID); e2 == nil && again != nil {
				return again, again.LeaseToken, nil
			}
			if active, e2 := s.Store.ActiveSession(); e2 == nil && active != nil {
				return nil, "", ErrActiveExists
			}
		}
		return nil, "", err
	}
	_ = s.Store.InsertEvent(row.ID, ts, "created", "{}")
	return row, lease, nil
}

func (s *Service) Get(id string) (*store.SessionRow, error) {
	row, err := s.Store.GetSession(id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}
	return row, nil
}

func (s *Service) Current() (*store.SessionRow, error) {
	return s.Store.ActiveSession()
}

func (s *Service) SaveRecord(id, lease string, expected int, rec Record) (*store.SessionRow, error) {
	row, err := s.mustOwn(id, lease)
	if err != nil {
		return nil, err
	}
	if row.State != "collecting" {
		return nil, ErrBadState
	}
	if row.Paused {
		return nil, ErrPaused
	}
	if row.Revision != expected {
		return nil, ErrConflict
	}
	if err := ValidateRecord(&rec); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return nil, err
	}
	row.RecordJSON = string(raw)
	row.Step = rec.Step
	row.Revision++
	row.UpdatedAt = s.now().Format(time.RFC3339)
	if err := s.Store.SaveSession(row, expected); err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return row, nil
}

func (s *Service) Lock(id, lease string, expected int) (*store.SessionRow, error) {
	row, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if row.State == "locked" {
		return row, nil
	}
	if _, err := s.mustOwn(id, lease); err != nil {
		return nil, err
	}
	if row.State != "collecting" {
		return nil, ErrBadState
	}
	if row.Paused {
		return nil, ErrPaused
	}
	if row.Revision != expected {
		return nil, ErrConflict
	}
	ts := s.now().Format(time.RFC3339)
	row.LockedRecordJSON = row.RecordJSON
	row.State = "locked"
	row.Step = 5
	row.LockedAt = ts
	row.AltsAuthorizedAt = ts
	row.Revision++
	row.UpdatedAt = ts
	if err := s.Store.SaveSession(row, expected); err != nil {
		if errors.Is(err, store.ErrConflict) {
			again, _ := s.Get(id)
			if again != nil && again.State == "locked" {
				return again, nil
			}
			return nil, ErrConflict
		}
		return nil, err
	}
	_ = s.Store.InsertEvent(row.ID, ts, "locked", "{}")
	return row, nil
}

func (s *Service) Tentative(id, lease string, expected int, choice string, conf *int) (*store.SessionRow, error) {
	row, err := s.mustOwn(id, lease)
	if err != nil {
		return nil, err
	}
	if row.State != "locked" {
		return nil, ErrBadState
	}
	if row.Paused {
		return nil, ErrPaused
	}
	if row.Revision != expected {
		return nil, ErrConflict
	}
	choice = strings.ToUpper(choice)
	if !validPos(choice) {
		return nil, ErrValidation
	}
	if err := validConf(conf); err != nil {
		return nil, err
	}
	row.TentativeChoice = choice
	row.TentativeConfidence = conf
	row.Revision++
	row.UpdatedAt = s.now().Format(time.RFC3339)
	if err := s.Store.SaveSession(row, expected); err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return row, nil
}

func (s *Service) Confirm(id, lease string, expected int, choice string, conf *int) (*store.SessionRow, error) {
	row, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	choice = strings.ToUpper(choice)
	if !validPos(choice) {
		return nil, ErrValidation
	}
	if err := validConf(conf); err != nil {
		return nil, err
	}
	if row.State == "completed" {
		if row.ConfirmedChoice == choice {
			return row, nil
		}
		return nil, ErrChoiceConflict
	}
	if _, err := s.mustOwn(id, lease); err != nil {
		return nil, err
	}
	if row.State != "locked" {
		return nil, ErrBadState
	}
	if row.Paused {
		return nil, ErrPaused
	}
	if row.Revision != expected {
		return nil, ErrConflict
	}
	sha := shaAt(row, choice)
	hit := 0
	if sha == row.TargetSHA {
		hit = 1
	}
	ts := s.now().Format(time.RFC3339)
	row.ConfirmedChoice = choice
	row.ConfirmedConfidence = conf
	row.Hit = &hit
	row.State = "completed"
	row.Step = 6
	row.CompletedAt = ts
	row.LeaseToken = ""
	row.LeaseUntil = ""
	row.Revision++
	row.UpdatedAt = ts
	if err := s.Store.SaveSession(row, expected); err != nil {
		if errors.Is(err, store.ErrConflict) {
			again, _ := s.Get(id)
			if again != nil && again.State == "completed" && again.ConfirmedChoice == choice {
				return again, nil
			}
			if again != nil && again.State == "completed" {
				return nil, ErrChoiceConflict
			}
			return nil, ErrConflict
		}
		return nil, err
	}
	_ = s.Store.InsertEvent(row.ID, ts, "confirmed", `{"choice":"`+choice+`"}`)
	return row, nil
}

func (s *Service) Abandon(id, lease string, expected int) (*store.SessionRow, error) {
	row, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if row.State == "abandoned" {
		return row, nil
	}
	if row.State == "completed" {
		return nil, ErrBadState
	}
	if _, err := s.mustOwn(id, lease); err != nil {
		return nil, err
	}
	if row.Revision != expected {
		return nil, ErrConflict
	}
	ts := s.now().Format(time.RFC3339)
	row.AbandonedAfterAlts = row.State == "locked" || row.AltsAuthorizedAt != ""
	row.State = "abandoned"
	row.AbandonedAt = ts
	row.LeaseToken = ""
	row.LeaseUntil = ""
	row.Revision++
	row.UpdatedAt = ts
	if err := s.Store.SaveSession(row, expected); err != nil {
		if errors.Is(err, store.ErrConflict) {
			again, _ := s.Get(id)
			if again != nil && again.State == "abandoned" {
				return again, nil
			}
			return nil, ErrConflict
		}
		return nil, err
	}
	_ = s.Store.InsertEvent(row.ID, ts, "abandoned", "{}")
	return row, nil
}

func (s *Service) Comment(id, text string) (*store.SessionRow, error) {
	if err := boundText("comment", text); err != nil {
		return nil, err
	}
	row, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if row.State != "completed" {
		return nil, ErrBadState
	}
	expected := row.Revision
	ts := s.now().Format(time.RFC3339)
	row.Comment = text
	row.CommentUpdatedAt = ts
	row.UpdatedAt = ts
	if err := s.patch(row, expected); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) Heartbeat(id, token string, transfer bool) (string, *store.SessionRow, error) {
	row, err := s.Get(id)
	if err != nil {
		return "", nil, err
	}
	if row.State != "collecting" && row.State != "locked" {
		return "", row, nil
	}
	now := s.now()
	held := row.LeaseToken != "" && row.LeaseUntil != ""
	until, _ := time.Parse(time.RFC3339, row.LeaseUntil)
	expired := !held || until.Before(now)
	if !expired && token == row.LeaseToken {
		expected := row.Revision
		row.LeaseUntil = now.Add(leaseTTL).Format(time.RFC3339)
		row.UpdatedAt = now.Format(time.RFC3339)
		if err := s.Store.PatchSession(row, expected); err != nil {
			if errors.Is(err, store.ErrConflict) {
				again, gerr := s.Get(id)
				if gerr != nil {
					return "", nil, gerr
				}
				return token, again, nil
			}
			return "", nil, err
		}
		return row.LeaseToken, row, nil
	}
	if !expired && !transfer {
		return "", row, ErrLease
	}
	next, err := randomHex(s.reader(), 16)
	if err != nil {
		return "", nil, ErrRand
	}
	expected := row.Revision
	row.LeaseToken = next
	row.LeaseUntil = now.Add(leaseTTL).Format(time.RFC3339)
	row.Revision++
	row.UpdatedAt = now.Format(time.RFC3339)
	if err := s.patch(row, expected); err != nil {
		return "", nil, err
	}
	if transfer && !expired {
		_ = s.Store.InsertEvent(row.ID, now.Format(time.RFC3339), "lease_transfer", "{}")
	}
	return next, row, nil
}

func (s *Service) Timing(id, lease string, seq, deltaMs int) (*store.SessionRow, error) {
	row, err := s.mustOwn(id, lease)
	if err != nil {
		return nil, err
	}
	if seq <= row.TimingSeq {
		return row, nil
	}
	if seq != row.TimingSeq+1 {
		return nil, ErrConflict
	}
	if row.Paused || (row.State != "collecting" && row.State != "locked") {
		return row, nil
	}
	if deltaMs < 0 {
		deltaMs = 0
	}
	if deltaMs > maxTimingMs {
		deltaMs = maxTimingMs
	}
	expected := row.Revision
	if row.State == "collecting" {
		row.CollectionMs += deltaMs
	} else {
		row.ChoiceMs += deltaMs
	}
	row.TimingSeq = seq
	row.UpdatedAt = s.now().Format(time.RFC3339)
	if err := s.patch(row, expected); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) SetPaused(id, lease string, paused bool) (*store.SessionRow, error) {
	row, err := s.mustOwn(id, lease)
	if err != nil {
		return nil, err
	}
	if row.State != "collecting" && row.State != "locked" {
		return nil, ErrBadState
	}
	expected := row.Revision
	row.Paused = paused
	row.UpdatedAt = s.now().Format(time.RFC3339)
	if err := s.patch(row, expected); err != nil {
		return nil, err
	}
	kind := "resumed"
	if paused {
		kind = "paused"
	}
	_ = s.Store.InsertEvent(row.ID, row.UpdatedAt, kind, "{}")
	return row, nil
}

func (s *Service) RecordExample(id, lease string, step int) (*store.SessionRow, error) {
	row, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if row.State == "abandoned" {
		return nil, ErrBadState
	}
	if row.State == "collecting" || row.State == "locked" {
		if _, err := s.mustOwn(id, lease); err != nil {
			return nil, err
		}
	}
	expected := row.Revision
	var opened []int
	_ = json.Unmarshal([]byte(row.OpenedExamplesJSON), &opened)
	opened = append(opened, step)
	raw, _ := json.Marshal(opened)
	row.OpenedExamplesJSON = string(raw)
	row.UpdatedAt = s.now().Format(time.RFC3339)
	if err := s.patch(row, expected); err != nil {
		return nil, err
	}
	_ = s.Store.InsertEvent(row.ID, row.UpdatedAt, "example_opened", fmt.Sprintf(`{"step":%d}`, step))
	return row, nil
}

func (s *Service) patch(row *store.SessionRow, expected int) error {
	if err := s.Store.PatchSession(row, expected); err != nil {
		if errors.Is(err, store.ErrConflict) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (s *Service) RecordLoaded(id string, pos []string) error {
	row, err := s.Get(id)
	if err != nil {
		return err
	}
	if !ImagesAuthorized(row) {
		return ErrBadState
	}
	raw, _ := json.Marshal(pos)
	return s.Store.InsertEvent(row.ID, s.now().Format(time.RFC3339), "image_loaded", string(raw))
}

func (s *Service) mustOwn(id, lease string) (*store.SessionRow, error) {
	row, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if row.State != "collecting" && row.State != "locked" {
		return row, nil
	}
	now := s.now()
	until, err := time.Parse(time.RFC3339, row.LeaseUntil)
	if err != nil || row.LeaseToken == "" || until.Before(now) || lease != row.LeaseToken {
		return nil, ErrLease
	}
	return row, nil
}

func ImagesAuthorized(row *store.SessionRow) bool {
	if row == nil {
		return false
	}
	switch row.State {
	case "locked", "completed":
		return true
	case "abandoned":
		return row.AbandonedAfterAlts
	default:
		return false
	}
}

func SHAForPos(row *store.SessionRow, pos string) string {
	return shaAt(row, strings.ToUpper(pos))
}

func CorrectPos(row *store.SessionRow) string {
	for i, p := range []string{row.PosA, row.PosB, row.PosC, row.PosD} {
		if p == row.TargetSHA {
			return string(positions[i])
		}
	}
	return ""
}

func HoldsLease(row *store.SessionRow, token string, now time.Time) bool {
	if row == nil || token == "" || row.LeaseToken == "" || token != row.LeaseToken {
		return false
	}
	until, err := time.Parse(time.RFC3339, row.LeaseUntil)
	return err == nil && !until.Before(now)
}

func shaAt(row *store.SessionRow, pos string) string {
	switch pos {
	case "A":
		return row.PosA
	case "B":
		return row.PosB
	case "C":
		return row.PosC
	case "D":
		return row.PosD
	}
	return ""
}

func validPos(p string) bool {
	return p == "A" || p == "B" || p == "C" || p == "D"
}

func validConf(c *int) error {
	if c == nil {
		return nil
	}
	if *c < 0 || *c > 100 {
		return ErrValidation
	}
	return nil
}

func drawAssignment(r io.Reader, hashes []string) (target string, order [4]string, err error) {
	n := len(hashes)
	pick := make([]string, n)
	copy(pick, hashes)
	idx, err := randIntn(r, n)
	if err != nil {
		return "", order, ErrRand
	}
	target = pick[idx]
	pick = append(pick[:idx], pick[idx+1:]...)
	var dist [3]string
	for i := 0; i < 3; i++ {
		j, err := randIntn(r, len(pick))
		if err != nil {
			return "", order, ErrRand
		}
		dist[i] = pick[j]
		pick = append(pick[:j], pick[j+1:]...)
	}
	alts := []string{target, dist[0], dist[1], dist[2]}
	for i := 3; i > 0; i-- {
		j, err := randIntn(r, i+1)
		if err != nil {
			return "", order, ErrRand
		}
		alts[i], alts[j] = alts[j], alts[i]
	}
	copy(order[:], alts)
	seen := map[string]bool{}
	for _, h := range order {
		if h == "" || seen[h] {
			return "", order, ErrRand
		}
		seen[h] = true
	}
	return target, order, nil
}

func randIntn(r io.Reader, n int) (int, error) {
	if n <= 0 {
		return 0, ErrRand
	}
	if n == 1 {
		return 0, nil
	}
	max := (^uint64(0) / uint64(n)) * uint64(n)
	var buf [8]byte
	for {
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, err
		}
		v := binary.BigEndian.Uint64(buf[:])
		if v < max {
			return int(v % uint64(n)), nil
		}
	}
}

func randomHex(r io.Reader, n int) (string, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func randomCode(r io.Reader) (string, error) {
	a, err := randIntn(r, 9000)
	if err != nil {
		return "", err
	}
	b, err := randIntn(r, 9000)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d–%d", 1000+a, 1000+b), nil
}
