package service

import (
	"time"
	"github.com/jarrancarr/website"
	"net/http"
	//"fmt"
	"errors"
	"io/ioutil"
	"strings"
	"strconv"
)

var (
	Logger *website.Log
)

type Message struct {
	message,author,room string
	timestamp time.Time
	read bool
	reference *Message
}
type Room struct {
	passCode string
	message []*Message
	input chan *Message
	ears map[string] chan *Message
	Moderator []*website.Account
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

func CreateService(acs *website.AccountService) *MessageService {
	Logger.Trace.Println();
	mss := MessageService{make(map[string]*Room), acs}
	return &mss
}
func (mss *MessageService) Execute(data []string ,p *website.Page) string {
	Logger.Trace.Println();
	switch data[0] {
		case "roomList": return mss.roomList()
	}
	return "huh?"
}
func (mss *MessageService) Status() string {
	return "good"
}
func (mss *MessageService) createRoom(name, passCode string, user *website.Account) {
	Logger.Trace.Println();
	mss.room[name] = &Room{passCode, nil, make(chan *Message), make(map[string] chan *Message), []*website.Account{user}}
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
	m := Message{message, author, roomName, time.Now(), false, nil}
	room.message = append(room.message, &m)
	for _, ear := range(room.ears) {
		ear <- &m
	}
}
func (pmq PersonalMessageQueue) listen(ear <-chan *Message) {
	for m := range(ear) {
		pmq.messages = append(pmq.messages, m)
	}
}
func (mss *MessageService) roomList() string {
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
	return roomList
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
func (mss *MessageService) CreateRoomAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	Logger.Trace.Println("mss.AddRoomAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page)")
	httpData, _ :=ioutil.ReadAll(r.Body)
	if (httpData == nil || len(httpData) == 0) {
		return "", errors.New("No Data")
	}
	dataList := strings.Split(string(httpData),"&")
	roomName := strings.Split(dataList[0],"=")[1]
	roomPass := strings.Split(dataList[1],"=")[1]
	
	user, _ := mss.acs.GetAccount(s.GetUserName())
	mss.createRoom(roomName,roomPass,user)
	
	w.Write([]byte(mss.roomList()))
	return "ok", nil
}