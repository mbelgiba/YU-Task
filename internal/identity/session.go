package identity

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const (
	CookieName = "yu_dev_session"
	MaxAge     = 8 * time.Hour
)

type Session struct {
	UserID   string `json:"userId"`
	TenantID string `json:"tenantId"`
	Role     string `json:"role"`
	CSRF     string `json:"csrfToken"`
	Expires  time.Time
}

type Sessions struct {
	mu     sync.Mutex
	values map[string]Session
}

func NewSessions() *Sessions { return &Sessions{values: make(map[string]Session)} }

func (s *Sessions) Create(userID, tenantID, role string) (string, Session, error) {
	id, err := randomToken()
	if err != nil {
		return "", Session{}, err
	}
	csrf, err := randomToken()
	if err != nil {
		return "", Session{}, err
	}
	session := Session{UserID: userID, TenantID: tenantID, Role: role, CSRF: csrf, Expires: time.Now().Add(MaxAge)}
	s.mu.Lock()
	s.values[id] = session
	s.mu.Unlock()
	return id, session, nil
}

func (s *Sessions) Get(r *http.Request) (Session, bool) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return Session{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.values[cookie.Value]
	if !ok {
		return Session{}, false
	}
	if time.Now().After(session.Expires) {
		delete(s.values, cookie.Value)
		return Session{}, false
	}
	return session, true
}

func (s *Sessions) Delete(r *http.Request) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return
	}
	s.mu.Lock()
	delete(s.values, cookie.Value)
	s.mu.Unlock()
}

func randomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}
