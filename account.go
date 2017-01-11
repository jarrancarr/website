package website

import (
	//"fmt"
	"net/http"
	"errors"
)

type Permission struct {
	Name, Desc string
}

var StandardPermission = map[string]Permission{
	"admin":          Permission{"admin", "User can access administrator functions and pages"},
	"userManagement": Permission{"userManagement", "User can access user management"},
	"supervisor":     Permission{"supervisor", "User has supervisor resposibilities"},
	"basic":          Permission{"basic", "User has a basic user account"},
	"premium":        Permission{"premium", "User is a premium user"},
	"update":         Permission{"update", "User can update fields"},
}

type Role struct {
	Name, Desc string
	Permission []string
}

var StandardRoles = map[string]*Role{
	"admin":      &Role{"admin", "Administrator role", []string{"admin"}},
	"manager":    &Role{"manager", "Manager role", []string{"userManagement"}},
	"supervisor": &Role{"supervisor", "Supervisor role", []string{"supervisor"}},
	"basic":      &Role{"basic", "Basic Account", []string{"basic"}},
	"premium":    &Role{"premium", "Premium Account", []string{"premium", "basic"}},
}

type Account struct {
	Title, First, Middle, Last, Suffix, User, Password string
	role                                               []*Role
}

var Users = []Account{
	Account{"Mr.", "Jarran", "J", "Carr", "", "jcarr", "JCarr48", []*Role{StandardRoles["premium"]}},
	Account{"Mr.", "Jarran", "J", "Carr", "", "admin", "ADmin48", []*Role{StandardRoles["admin"]}},
	Account{"Mrs.", "Jamie", "N", "Carr", "", "jncarr", "JNCarr48", []*Role{StandardRoles["manager"]}},
	Account{"Mr.", "Logan", "J", "Carr", "", "lcarr", "LCarr48", []*Role{StandardRoles["basic"]}},
}

type AccountService struct {
	SiteSessionCookieName string
	Domain                string
	active 	map[string]*Session
}

func CreateAccountService() *AccountService {
	as := AccountService{"", "", make(map[string]*Session)}
	return &as
}

func (ecs *AccountService) Status() string {
	return "good"
}

func (acs *AccountService) Execute(data []string, page *Page) string {
	session := page.ActiveSession
	switch data[0] {
		case "getName":
			return session.Data["name"]
			break
		case "session":
			return session.Data[data[1]]
			break
		case "isLoggedIn":
			if session == nil {
				return "session is null"
			}
			if session.Data == nil {
				return "session.Data is null"
			}
			if session.Data["name"] == "Anonymous" {
				return "False"
			} else {
				return "True"
			}
		case "getStatus":
			return session.Data["status"]
			break
	}
	return ""
}

func (acs *AccountService) Get(page *Page, session *Session, data []string) Item {
	return struct { 
		Title , Name, Desc string
		} {"Duke","Bingo","The Man!"}
}

func (acs *AccountService) LoginPostHandler(w http.ResponseWriter, r *http.Request, s *Session, p *Page) (string, error) {
	userName := r.Form.Get("UserName")
	password := r.Form.Get("Password")

	for _, u := range Users {
		if userName == u.User && password == u.Password {
			s.Data["name"] = u.First + " " + u.Middle + ". " + u.Last
			s.Data["userName"] = u.User
			s.Item["account"] = u
			acs.active[u.User] = s
			return r.Form.Get("redirect"), nil
		}
	}
	return "login", nil
}

func (acs *AccountService) CheckSecure(w http.ResponseWriter, r *http.Request, s *Session, p *Page) (string, error) {
	if s.Data["name"] == "Anonymous" || s.Data["name"] == "" {
		if s != nil {
			s.Data["status"] = "User credentials not recognized."
		}
		return "login", errors.New("Invalid credentials!")
	}
	s.Data["status"] = "User access granted."
	return "ok", nil
}

func (acs *AccountService) GetUserSession(userName string) *Session {
	return acs.active[userName]
}
