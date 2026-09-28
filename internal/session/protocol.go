package session

import (
	"encoding/json"
	"fmt"
	"math"
	"unicode/utf8"
)

const (
	ProtocolVersion = "crv-record-v1"
	HelpVersion     = "crv-help-v1"
	MaxText         = 20000
	MaxRecordBytes  = 10 << 20
	MaxStrokes      = 4000
	MaxPoints       = 20000
	LogicalW        = 1000.0
	LogicalH        = 620.0
)

type Option struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type GroupDef struct {
	Key       string   `json:"key"`
	Title     string   `json:"title"`
	Collapsed bool     `json:"collapsed,omitempty"`
	Options   []Option `json:"options"`
}

type HelpEntry struct {
	Goal    string `json:"goal"`
	How     string `json:"how"`
	Example string `json:"example"`
	Care    string `json:"care"`
}

type ProtocolSnapshot struct {
	Version     string      `json:"version"`
	HelpVersion string      `json:"helpVersion"`
	GroupsI     []GroupDef  `json:"groupsI"`
	GroupsII    []GroupDef  `json:"groupsII"`
	Help        []HelpEntry `json:"help"`
}

type GroupValue struct {
	IDs  []string `json:"ids"`
	Note string   `json:"note"`
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Stroke struct {
	Points []Point `json:"points"`
	Width  float64 `json:"width"`
}

type Drawings struct {
	Ideogram []Stroke `json:"ideogram"`
	Sketch   []Stroke `json:"sketch"`
}

type Record struct {
	Version        string                `json:"version"`
	Disposition    string                `json:"disposition"`
	Concentration  string                `json:"concentration"`
	AttributesOpen bool                  `json:"attributesOpen"`
	Groups         map[string]GroupValue `json:"groups"`
	AOL1           string                `json:"aol1"`
	Sensory        string                `json:"sensory"`
	AOL2           string                `json:"aol2"`
	Forms          string                `json:"forms"`
	Dimensions     string                `json:"dimensions"`
	Positions      string                `json:"positions"`
	Spatial        string                `json:"spatial"`
	AOL3           string                `json:"aol3"`
	Drawings       Drawings              `json:"drawings"`
	Summary        []string              `json:"summary"`
	Confidence     *int                  `json:"confidence"`
	Step           int                   `json:"step"`
}

func SnapshotProtocol() ProtocolSnapshot {
	return ProtocolSnapshot{
		Version:     ProtocolVersion,
		HelpVersion: HelpVersion,
		GroupsI:     groupsI,
		GroupsII:    groupsII,
		Help:        helpEntries,
	}
}

func EmptyRecord() Record {
	return Record{
		Version: ProtocolVersion,
		Groups:  map[string]GroupValue{},
		Drawings: Drawings{
			Ideogram: []Stroke{},
			Sketch:   []Stroke{},
		},
		Summary: []string{"", "", "", "", ""},
		Step:    1,
	}
}

func ParseRecord(raw string) (Record, error) {
	if raw == "" {
		return EmptyRecord(), nil
	}
	var rec Record
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		return Record{}, fmt.Errorf("registro inválido")
	}
	if rec.Groups == nil {
		rec.Groups = map[string]GroupValue{}
	}
	if rec.Summary == nil {
		rec.Summary = []string{"", "", "", "", ""}
	}
	if rec.Drawings.Ideogram == nil {
		rec.Drawings.Ideogram = []Stroke{}
	}
	if rec.Drawings.Sketch == nil {
		rec.Drawings.Sketch = []Stroke{}
	}
	if rec.Step < 1 {
		rec.Step = 1
	}
	return rec, nil
}

func ValidateRecord(rec *Record) error {
	if rec.Version != "" && rec.Version != ProtocolVersion {
		return fmt.Errorf("versão de registro não suportada")
	}
	rec.Version = ProtocolVersion
	if err := boundText("disposition", rec.Disposition); err != nil {
		return err
	}
	switch rec.Concentration {
	case "", "Baixa", "Moderada", "Alta":
	default:
		return fmt.Errorf("concentração inválida")
	}
	if rec.Groups == nil {
		rec.Groups = map[string]GroupValue{}
	}
	allowed := allowedGroupIDs()
	for key, gv := range rec.Groups {
		ids, ok := allowed[key]
		if !ok {
			return fmt.Errorf("campo desconhecido")
		}
		seen := map[string]bool{}
		for _, id := range gv.IDs {
			if !ids[id] {
				return fmt.Errorf("opção desconhecida")
			}
			if seen[id] {
				return fmt.Errorf("opção duplicada")
			}
			seen[id] = true
		}
		if err := boundText(key, gv.Note); err != nil {
			return err
		}
	}
	for _, pair := range []struct{ name, v string }{
		{"aol1", rec.AOL1}, {"sensory", rec.Sensory}, {"aol2", rec.AOL2},
		{"forms", rec.Forms}, {"dimensions", rec.Dimensions}, {"positions", rec.Positions},
		{"spatial", rec.Spatial}, {"aol3", rec.AOL3},
	} {
		if err := boundText(pair.name, pair.v); err != nil {
			return err
		}
	}
	if rec.Confidence != nil {
		if *rec.Confidence < 0 || *rec.Confidence > 100 {
			return fmt.Errorf("confiança fora do intervalo")
		}
	}
	if rec.Summary == nil {
		rec.Summary = []string{"", "", "", "", ""}
	}
	if len(rec.Summary) > 5 {
		return fmt.Errorf("no máximo cinco características")
	}
	for len(rec.Summary) < 5 {
		rec.Summary = append(rec.Summary, "")
	}
	for i, s := range rec.Summary {
		if err := boundText("summary", s); err != nil {
			return err
		}
		rec.Summary[i] = s
	}
	if rec.Step < 1 || rec.Step > 4 {
		rec.Step = 1
	}
	if err := validateDrawing("ideogram", rec.Drawings.Ideogram); err != nil {
		return err
	}
	if err := validateDrawing("sketch", rec.Drawings.Sketch); err != nil {
		return err
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("registro inválido")
	}
	if len(raw) > MaxRecordBytes {
		return fmt.Errorf("registro excede 10 MiB")
	}
	return nil
}

func boundText(name, s string) error {
	if !utf8.ValidString(s) {
		return fmt.Errorf("%s inválido", name)
	}
	if utf8.RuneCountInString(s) > MaxText {
		return fmt.Errorf("%s excede 20000 caracteres", name)
	}
	return nil
}

func validateDrawing(name string, strokes []Stroke) error {
	if len(strokes) > MaxStrokes {
		return fmt.Errorf("%s tem traços demais", name)
	}
	total := 0
	for _, st := range strokes {
		if st.Width < 1 || st.Width > 20 || math.IsNaN(st.Width) || math.IsInf(st.Width, 0) {
			return fmt.Errorf("espessura inválida")
		}
		if len(st.Points) == 0 || len(st.Points) > MaxPoints {
			return fmt.Errorf("traço inválido")
		}
		total += len(st.Points)
		if total > MaxPoints*2 {
			return fmt.Errorf("%s tem pontos demais", name)
		}
		for _, p := range st.Points {
			if math.IsNaN(p.X) || math.IsNaN(p.Y) || math.IsInf(p.X, 0) || math.IsInf(p.Y, 0) {
				return fmt.Errorf("coordenada inválida")
			}
			if p.X < 0 || p.X > LogicalW || p.Y < 0 || p.Y > LogicalH {
				return fmt.Errorf("coordenada fora da área")
			}
		}
	}
	return nil
}

func allowedGroupIDs() map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, g := range append(append([]GroupDef{}, groupsI...), groupsII...) {
		m := map[string]bool{}
		for _, o := range g.Options {
			m[o.ID] = true
		}
		out[g.Key] = m
	}
	return out
}

func opt(idsLabels ...string) []Option {
	out := make([]Option, 0, len(idsLabels)/2)
	for i := 0; i+1 < len(idsLabels); i += 2 {
		out = append(out, Option{ID: idsLabels[i], Label: idsLabels[i+1]})
	}
	return out
}

var groupsI = []GroupDef{
	{Key: "movement", Title: "Movimento / forma", Options: opt(
		"horizontal", "Horizontal", "vertical", "Vertical", "diagonal", "Diagonal",
		"curved", "Curvo", "wavy", "Ondulado", "angular", "Angular",
		"circular", "Circular", "up", "Sobe", "down", "Desce")},
	{Key: "feel", Title: "Sensação / consistência", Options: opt(
		"solid", "Sólido", "hard", "Duro", "soft", "Macio", "fluid", "Fluido",
		"gaseous", "Gasoso", "smooth", "Liso", "rough", "Áspero", "granular", "Granulado")},
	{Key: "gestalt", Title: "Categoria geral", Options: opt(
		"water", "Água", "terrain", "Terreno / relevo", "structure", "Estrutura construída",
		"vegetation", "Vegetação", "living", "Ser vivo", "undefined", "Indefinido")},
}

var groupsII = []GroupDef{
	{Key: "colors", Title: "Cores", Options: opt(
		"white", "Branco", "black", "Preto", "gray", "Cinza", "blue", "Azul", "green", "Verde",
		"red", "Vermelho", "yellow", "Amarelo", "brown", "Marrom", "orange", "Laranja", "violet", "Violeta")},
	{Key: "light", Title: "Luminosidade", Options: opt(
		"light", "Claro", "dark", "Escuro", "bright", "Brilhante", "matte", "Fosco",
		"reflective", "Reflexivo", "translucent", "Translúcido")},
	{Key: "texture", Title: "Textura", Options: opt(
		"smooth", "Liso", "rough", "Áspero", "granular", "Granulado", "fibrous", "Fibroso",
		"sticky", "Pegajoso", "slippery", "Escorregadio")},
	{Key: "temp", Title: "Temperatura", Options: opt("cold", "Frio", "cool", "Fresco", "warm", "Morno", "hot", "Quente")},
	{Key: "wet", Title: "Umidade", Options: opt("dry", "Seco", "damp", "Úmido", "wet", "Molhado")},
	{Key: "sound", Title: "Sons", Collapsed: true, Options: opt(
		"silent", "Silencioso", "continuous", "Contínuo", "intermittent", "Intermitente",
		"low", "Grave", "high", "Agudo", "rhythmic", "Rítmico", "water", "Ruído de água",
		"mechanical", "Mecânico", "voices", "Vozes")},
	{Key: "smell", Title: "Cheiros", Collapsed: true, Options: opt(
		"earthy", "Terroso", "vegetal", "Vegetal", "saline", "Salino", "floral", "Floral",
		"chemical", "Químico", "burnt", "Queimado")},
	{Key: "taste", Title: "Sabores", Collapsed: true, Options: opt(
		"sweet", "Doce", "salty", "Salgado", "bitter", "Amargo", "sour", "Ácido", "metallic", "Metálico")},
}

var helpEntries = []HelpEntry{
	{Goal: "Preparar uma sessão com um alvo oculto.", How: "Encontre um ambiente confortável. O código identifica a sessão, sem indicar o conteúdo do alvo. O tempo é apenas uma referência.", Example: "Você pode deixar disposição e concentração em branco e iniciar quando estiver pronto.", Care: "O cronômetro orienta; não encerra a sessão."},
	{Goal: "Registrar um gesto breve e uma impressão geral.", How: "Desenhe com o mouse. Em Registrar impressões, anote movimento, sensação e uma categoria, se surgir. Checkboxes e textos são opcionais.", Example: "Movimento: curvo. Sensação: fluido. Categoria: água? É um exemplo de preenchimento, não uma regra de interpretação.", Care: "Um traço ondulado não significa obrigatoriamente água. Se não surgir uma interpretação, deixe indefinido."},
	{Goal: "Registrar características sensoriais simples.", How: "Marque atributos, escreva livremente ou combine os dois. Os campos vazios significam não registrado, não ausente.", Example: "Frio, áspero e cinza são descritores. Parece um castelo é uma identificação: registre em Hipóteses / AOL.", Care: "Não preencha para completar a lista. As sugestões são sempre iguais, independentemente do alvo."},
	{Goal: "Representar formas, dimensões e posições.", How: "Use linhas simples. Anote tamanho relativo e localização dos elementos. A borracha remove um traço inteiro.", Example: "Um retângulo alto à esquerda de uma área ampla comunica uma relação espacial sem identificar o objeto.", Care: "Não precisa saber desenhar. O esboço é separado do ideograma inicial."},
	{Goal: "Conferir e finalizar o registro original.", How: "Revise também as hipóteses e contradições. Você pode voltar aos estágios anteriores enquanto o registro está aberto.", Example: "Você pode destacar até cinco características, deixando as restantes em branco.", Care: "Depois de mostrar as alternativas, a edição original será bloqueada. Comentários posteriores ficam separados."},
	{Goal: "Escolher a imagem mais compatível com seu registro.", How: "Leia o registro completo e examine as quatro alternativas. A seleção pode mudar até clicar em Confirmar escolha.", Example: "Se uma imagem combina com uma cor, mas contradiz seu desenho, considere as duas informações.", Care: "Não reinterprete o registro para ajustá-lo à imagem. Sua escolha ficará definitiva após a confirmação."},
	{Goal: "Comparar e registrar uma reflexão posterior.", How: "Confira a escolha e o alvo correto. Escreva o que observou no comentário pós-feedback, sem mudar o original.", Example: "Anote correspondências e divergências específicas, preservando o que escreveu antes.", Care: "Um acerto isolado não demonstra desempenho acima do acaso."},
}
