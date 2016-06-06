package service

type AccountService struct {
	Session       map[string]*Session
	SessionCookie string
}

type Account struct {
	name string
}

type Session struct {
	user *Account
	item map[string]interface{}
}

func CreateAccountService() Service {
	as := AccountService{make(map[string]*Session), generateSessionKey()}
	return &as
}

func (ecs *AccountService) Status() string {
	return "good"
}

func (acs *AccountService) Execute(command, user, data string) string {
	switch command {
	case "Login":
		acs.CreateSession(&Account{user})
		return "user created"
	case "Logout":
		acs.RemoveSession(user, data)
		return "user logged off"
	}
	return ""
}

func (acs *AccountService) GetSession(key string) *Session {
	return acs.Session[key]
}

func (acs *AccountService) CreateSession(user *Account) string {
	key := generateSessionKey()
	acs.Session[key] = &Session{user, make(map[string]interface{})}
	return key
}

func (acs *AccountService) RemoveSession(user *Account) string {

func generateSessionKey() string {
	return "test"
}

func Session(w http.ResponseWriter, r *http.Request) {
	
}
