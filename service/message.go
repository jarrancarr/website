package service

type MessageService struct {
	message map[string]string
}

func CreateMessageService() Service {
	ms := MessageService{nil}
	return &ms
}

func (ecs *MessageService) Status() string {
	return "good"
}

func (ecs *MessageService) Execute(user, command, data string) string {
	return "message service executed commmand " + command + " with data " + data
}
