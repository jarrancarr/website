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

func (ecs *MessageService) Execute(service *website.Session, data []string) string {
	return "message service executed commmand " + data[0] + " with data " + data[1]
}
