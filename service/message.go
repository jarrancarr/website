package service

import (
	"time"
	"github.com/jarrancarr/website"
	"net/http"
	"fmt"
	"io/ioutil"
	"strings"
	"strconv"
)

type Message struct {
	message,author,room string
	timestamp time.Time
	read bool
}
type Room struct {
	passCode string
	message []Message
	ears map[string] chan *Message
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

func CreateMessageService(acs *website.AccountService) *MessageService {
	mss := MessageService{make(map[string]*Room), acs}
	return &mss
}
func (mss *MessageService) Execute(s *website.Session, data []string) string {
//	switch data[1] {
//	}
	return "huh?"
}
func (mss *MessageService) Status() string {
	return "good"
}
func (mss *MessageService) createRoom(name, passCode, userName string) {
	mss.room[name] = &Room{passCode, make([]Message,100), make(map[string] chan *Message,100), "", ""}
	//mss.join(name, userName)
}
func (mss *MessageService) join(roomName, userName string) {
	ear := make(chan *Message)
	user := mss.acs.GetUserSession(userName)
	mss.room[roomName].ears[userName] = ear
	pmq := PersonalMessageQueue{make([]*Message,100), 0, 0}
	user.AddItem("MessageService-Queue-"+roomName, pmq)
	go pmq.listen(ear)
}
func (mss *MessageService) exit(roomName, userName string) {
	user := mss.acs.GetUserSession(userName)
	mss.room[roomName].ears[user.Data["userName"]] = nil
	user.Item["MessageService-Queue-"+roomName] = nil
}
func (room *Room) post(message, author, roomName string) {
	m := Message{message, author, roomName, time.Now(), false}
	room.message = append(room.message, m)
	for _, ear := range(room.ears) {
		ear <- &m
	}
}
func (pmq PersonalMessageQueue) listen(ear <-chan *Message) {
	for m := range(ear) {
		pmq.messages = append(pmq.messages, m)
	}
}

func (mss *MessageService) TestAJAX(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	w.Write([]byte(`{ "one": "Singular sensation", "two": "Beady little eyes", "three": "Little birds pitch by my doorstep"}`))
	return "ok", nil
}
func (mss *MessageService) Get(page *website.Page, session *website.Session, data []string) website.Item {
	switch data[0] {
	}
	t := "Duke"
	n := "Bingo"
	d := "The Man!"
	return struct {
			Title, Name, Desc string
		} {
			t, n, d,
		}
}

func (mss *MessageService) PostStatement(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	w.Write([]byte(`{ "one": "Singular sensation", "two": "Beady little eyes", "three": "Little birds pitch by my doorstep"}`))
	return "ok", nil
}
func (mss *MessageService) CreateRoom(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	httpData, _ :=ioutil.ReadAll(r.Body)
	requestedRoom := strings.Split(string(httpData),"=")[1]
	mss.createRoom(requestedRoom,"password",s.GetUserName())
	fmt.Println("create room: "+requestedRoom)
	roomList := "{"
	first := true
	for k,r := range(mss.room) {
		if !first {
			roomList += ","
		} else {
			first = false
		}
		roomList += `"`+k+`":`+strconv.Itoa(len(r.ears))
	}
	roomList += "}"
	fmt.Println(roomList)
	w.Write([]byte(roomList))
	return "ok", nil
}
