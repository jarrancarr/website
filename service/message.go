package service

import (
	"github.com/jarrancarr/website"
)

type MessageService struct {
	message map[string]string
}

func CreateMessageService() website.Service {
	ms := MessageService{nil}
	return &ms
}

func (ecs *MessageService) Status() string {
	return "good"
}

func (ecs *MessageService) Execute(command, user string, data []string) string {
	return "message service executed commmand " + command + " with data " + data[0]
}
