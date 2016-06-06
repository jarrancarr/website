package service

import (
	"net/http"
	"time"
)

type Account struct {
	name string
}

type Session struct {
	user *Account
	item map[string]interface{}
}

type AccountService struct {
	SiteSessionCookieName string
	Domain                string
	session               map[string]*Session
}

func CreateAccountService() *AccountService {
	as := AccountService{generateSessionKey(), "", make(map[string]*Session)}
	return &as
}

func (ecs *AccountService) Status() string {
	return "good"
}

func (acs *AccountService) Execute(command, user, sessionKey string) string {
	switch command {
	case "Login":
		if acs.session == nil {
			acs.session = make(map[string]*Session)
		}
		acs.session[sessionKey].user = &Account{user}
		return "user created"
	case "Logout":
		acs.RemoveSession(sessionKey)
		return "user logged off"
	}
	return ""
}

func (acs *AccountService) GetSession(key string) *Session {
	return acs.session[key]
}

func (acs *AccountService) CreateSession() string {
	key := generateSessionKey()
	acs.session[key] = &Session{nil, make(map[string]interface{})}
	return key
}

func (acs *AccountService) RemoveSession(key string) {
	acs.session[key] = nil
}

func generateSessionKey() string {
	return "test"
}

func (acs *AccountService) Action(w http.ResponseWriter, r *http.Request) string {
	return ""
}

func (acs *AccountService) ValidateSession(w http.ResponseWriter, r *http.Request) {
	sessionCookie, err := r.Cookie(acs.SiteSessionCookieName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if sessionCookie == nil {
		key := acs.CreateSession()
		expire := time.Now().Add(time.Minute * 20)
		cookie := http.Cookie{Name: acs.SiteSessionCookieName, Value: key, Path: "/", Domain: acs.Domain, Expires: expire}
		http.SetCookie(w, &cookie)
	}
}
