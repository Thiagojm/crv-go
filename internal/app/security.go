package app

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type Security struct {
	mu            sync.Mutex
	sessionSecret string
	csrfSecret    string
	bootstrap     string
	bootstrapUsed bool
}

func NewSecurity() (*Security, error) {
	s := &Security{}
	var err error
	if s.sessionSecret, err = randomHex(32); err != nil {
		return nil, err
	}
	if s.csrfSecret, err = randomHex(32); err != nil {
		return nil, err
	}
	if s.bootstrap, err = randomHex(32); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Security) BootstrapToken() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bootstrap
}

func (s *Security) ConsumeBootstrap(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bootstrapUsed || token == "" {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.bootstrap)) != 1 {
		return false
	}
	s.bootstrapUsed = true
	s.bootstrap = ""
	return true
}

func (s *Security) SessionCookieValue() string { return s.sessionSecret }
func (s *Security) CSRFToken() string          { return s.csrfSecret }

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func isLoopbackHost(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func originAllowed(r *http.Request, port int) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	p := strconv.Itoa(port)
	switch origin {
	case "http://127.0.0.1:"+p, "http://localhost:"+p, "http://[::1]:"+p:
		return true
	default:
		return false
	}
}

func cookieOK(r *http.Request, want string) bool {
	c, err := r.Cookie("crv_session")
	if err != nil || c.Value == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(c.Value), []byte(want)) == 1
}

func csrfOK(r *http.Request, want string) bool {
	got := r.Header.Get("X-CSRF-Token")
	if got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
