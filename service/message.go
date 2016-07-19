package service

import (
	"time"
	"github.com/jarrancarr/website"
)

type Message struct {
	message string
	author string // userName
	timestamp Time
	read bool
}

type Room struct {
	passCode string
	message []Message
	ears []chan *Message
	primeModerator string 		// userName
	assistantModerator string 		// userName
}

type MessageService struct {
	room map[string]*Room
	acs *website.AccountService
}

type PersonalMessageQueue struct {
	messages []*Message
	lastRead int
	lastPost int
}

func CreateMessageService(acs *website.AccountService) website.Service {
	mss := MessageService{make(map[string]*Room), acs}
	return &mss
}

func (mss *MessageService) Execute(s *website.Session, data []string) string {
	//fmt.Println("Command: " + command + " for user:" + user + " with data:" + data)
	switch data[1] {
		case "createRoom":
			ecs.room[data[2]] = &Room{data[3], make(Message,100), make(string,100), s.Data["userName"], ""}
			ecs.room[data[2]].join(s)
			return "ok"
		case "join":
			ecs.room[data[2]].join(data[2], s)
			return "ok"
		case "exit":
			return "ok"
		case "postMessage":
			mss.room[data[2]].post(data[3], s);	
			mss.disseminate(mss.room[data[2]].getLastMessage(), data[2])
			return "ok"
	}
	return "huh?"
}

func (ecs *MessageService) Status() string {
	return "good"
}

func (room *Room) join(name string, s *website.Session) {
	ear := make(chan *Message)
	room.rollCall = append(room.rollCall, &ear)
	pmq := PersonalMessageQueue{make([]*Message,100), 0, 0}
	s.Item["MessageService-Queue-"+name] = pmq
	go pmq.listen(ear)
}

func (room *Room) exit(name string, s *website.Session) {
	for idx, rc := range(room.rollCall) {
		if rc == name {
			room.rollCall = append(room.rollCall[:idx], room.rollCall[idx+1:])
		}
	}
	s.Item["MessageService-Queue-"+name] = nil
}

func (room *Room) post(message string, s *website.Session) {
	m := Message{message, s.Data["userName"], time.Now(), false}
	room.message = append(room.message, m)
	for ear := range(room.ears) {
		*ear <- &m
	}
}

func (pmq PersonalMessageQueue) listen(ear <-chan *Message) {
	for m := range(ear) {
		pmq.messages = append(q.messages, message)
	}
}
