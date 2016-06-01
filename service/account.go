package service

type AccountService struct {
	message map[string]string
}

func CreateAccountService() Service {
	as := AccountService{nil}
	return &as
}

func (ecs *AccountService) Status() string {
	return "good"
}

func (acs *AccountService) Execute(user, command, data string) string {
	return "message service executed commmand " + command + " with data " + data
}
