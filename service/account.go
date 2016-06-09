package service

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jarrancarr/website"
)

type AccountService struct {
	SiteSessionCookieName string
	Domain                string
	session               map[string]*website.Session
}

func CreateAccountService() *AccountService {
	as := AccountService{generateSessionKey(), "", make(map[string]*website.Session)}
	return &as
}

func (ecs *AccountService) Status() string {
	return "good"
}

func (acs *AccountService) Execute(command, user string, data []string) string {
	//fmt.Println("Command: " + command + " for user:" + user + " with data:" + data)
	switch command {
	case "Login":
		sessionKey := data[0]
		if acs.session == nil {
			acs.session = make(map[string]*website.Session)
		}
		if acs.session[sessionKey] == nil {
			acs.session[sessionKey] = &website.Session{}
		}
		acs.session[sessionKey].User = &website.Account{user}
		return "user created"
	case "Logout":
		sessionKey := data[0]
		acs.RemoveSession(sessionKey)
		return "user logged off"
	}
	return ""
}

func (acs *AccountService) GetSession(key string) *website.Session {
	return acs.session[key]
}

func (acs *AccountService) CreateSession() string {
	key := generateSessionKey()
	acs.session[key] = &website.Session{nil, make(map[string]interface{})}
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

func (acs *AccountService) ValidateSession(w http.ResponseWriter, r *http.Request, s *website.Session) string {
	sessionCookie, err := r.Cookie(acs.SiteSessionCookieName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if sessionCookie == nil {
		key := acs.CreateSession()
		expire := time.Now().Add(time.Minute * 20)
		cookie := http.Cookie{Name: acs.SiteSessionCookieName, Value: key, Path: "/", Domain: acs.Domain, Expires: expire}
		http.SetCookie(w, &cookie)
	} else {
		fmt.Println("Found session: " + sessionCookie.Value)
	}
	return "ok"
}

func (acs *AccountService) LoginPostHandler(w http.ResponseWriter, r *http.Request, s *website.Session) string {

	w.Write([]byte(acs.Execute("Login", r.FormValue("UserName"), []string{r.FormValue("Password")})))
	return "ok"
}

func (acs *AccountService) getSession(r *http.Request) (*website.Session, error) {
	sessionCookie, err := r.Cookie(acs.SiteSessionCookieName)
	if err != nil {
		return nil, err
	}
	return acs.session[sessionCookie.Value], nil
}
