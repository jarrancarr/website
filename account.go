package website

import (
	"fmt"
	"net/http"
)

type AccountService struct {
	SiteSessionCookieName string
	Domain                string
}

func CreateAccountService() *AccountService {
	as := AccountService{"", ""}
	return &as
}

func (ecs *AccountService) Status() string {
	return "good"
}

func (acs *AccountService) Execute(session *Session, data []string) string {
	//fmt.Println("Command: " + command + " for user:" + user + " with data:" + data)
	switch data[1] {
	case "Login":
	case "Logout":
	case "getName":
		return session.Data["name"]
		break
	}
	return ""
}

func (acs *AccountService) LoginPostHandler(w http.ResponseWriter, r *http.Request, s *Session) (string, error) {
	fmt.Println("Logging in...")
	userName := r.Form.Get("UserName")
	password := r.Form.Get("Password")
	if userName == "jcarr" && password == "JCarr48" {
		s.Data["name"] = "Jarran J Carr"
	}
	return "AccountService LoginPostHandler says ok", nil
}
