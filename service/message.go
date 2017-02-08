package service

import (
	"time"
	"github.com/jarrancarr/website"
	"net/http"
	"net/url"
	//"fmt"
	"errors"
	"io/ioutil"
	"strings"
	"sync"
	//"strconv"
)

var (
	Logger *website.Log
)

type Message struct {
	message,author string
	timestamp time.Time
	read bool
	reference *Message
}
type Room struct {
	passCode string
	message []*Message
	lock	sync.Mutex
}
type MessageService struct {
	room map[string]*Room
	acs *website.AccountService
	lock sync.Mutex
}
type PersonalMessageQueue struct {
	messages []*Message
	lastRead int
	lastPost int
}

func CreateService(acs *website.AccountService) *MessageService {
	Logger.Trace.Println();
	mss := MessageService{make(map[string]*Room), acs, sync.Mutex{}}
	return &mss
}
func (mss *MessageService) Execute(data []string ,p *website.Page) string {
	mss.lock.Lock()
	Logger.Trace.Println(data[0]+" "+data[1]);
	switch data[0] {
		case "roomList": return mss.roomList()
		case "addRoom": 
			if (mss.room[data[1]] == nil) {
				mss.room[data[1]] = &Room{"", make([]*Message,0), sync.Mutex{}}
			}
			mss.lock.Unlock()
			return ""
		case "post": 
			if (mss.room[data[1]] != nil) {
				mss.room[data[1]].post(data[2], data[3])
			}
			mss.lock.Unlock()
			return ""
	}
	mss.lock.Unlock()
	return "huh?"
}
func (mss *MessageService) Status() string {
	return "good"
}
func (mss *MessageService) createRoom(name, passCode string, user *website.Account) {
	Logger.Trace.Println("mss.createRoom("+name+","+passCode+","+user.User+")");
	mss.room[name] = &Room{passCode, make([]*Message,0), sync.Mutex{}}
}
func (mss *MessageService) join(roomName, userName string) {
	ear := make(chan *Message)
	user := mss.acs.GetUserSession(userName)
	//mss.room[roomName].ears[userName] = ear
	pmq := PersonalMessageQueue{make([]*Message,100), 0, 0}
	user.AddItem("MessageService-Queue-"+roomName, pmq)
	go pmq.listen(ear)
}
func (mss *MessageService) exit(roomName, userName string) {
	user := mss.acs.GetUserSession(userName)
	//mss.room[roomName].ears[user.Data["userName"]] = nil
	user.Item["MessageService-Queue-"+roomName] = nil
}
func (room *Room) post(author, message string) {
	room.lock.Lock()
	m := Message{message, author, time.Now(), false, nil}
	room.message = append(room.message, &m)
	room.lock.Unlock()
}
func (room *Room) getDiscussion(userName string) string {
	Logger.Debug.Println("room.getDiscussion("+userName+")")
	room.lock.Lock()
	discussion := "["
	first := true
	for _, m := range(room.message) {
		if !first {
			discussion += ","
		} else {
			first = false
		}
		if m.author == userName {
			discussion += `{"author":"","message":"`+url.QueryEscape(m.message)+`"}`
		} else {
			discussion += `{"author":"`+m.author+`","message":"`+url.QueryEscape(m.message)+`"}`
		}
	}
	discussion += "]"
	room.lock.Unlock()
	return discussion
}
func (pmq PersonalMessageQueue) listen(ear <-chan *Message) {
	for m := range(ear) {
		pmq.messages = append(pmq.messages, m)
	}
}
func (mss *MessageService) roomList() string {
	roomList := "{"
	first := true
	for k, _ := range(mss.room) {
		if !first {
			roomList += ","
		} else {
			first = false
		}
		roomList += `"`+k+`":1` //+strconv.Itoa(len(r.ears))
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
	
	Logger.Debug.Println("User is: "+p.ActiveSession.GetUserName())
	user, err := mss.acs.GetAccount(p.ActiveSession.GetUserName())
	if err != nil {
		Logger.Error.Println("No User "+p.ActiveSession.GetUserName()+" found")
		w.Write([]byte("ERROR"))
		return "NoUser", err
	}
	mss.createRoom(roomName,roomPass,user)
	
	w.Write([]byte(mss.roomList()))
	return "ok", nil
}
func (mss *MessageService) GetRoomsAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	Logger.Trace.Println("mss.GetRoomsAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page)")
	w.Write([]byte(mss.roomList()))
	return "ok", nil
}
func (mss *MessageService) MessageAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	Logger.Trace.Println("mss.MessageAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page)")
	httpData, _ :=ioutil.ReadAll(r.Body)
	if (httpData == nil || len(httpData) == 0) {
		return "", errors.New("No Data")
	}
	dataList := strings.Split(string(httpData),"&")
	roomName := strings.Split(dataList[0],"=")[1]
	message, _ := url.QueryUnescape(strings.Split(dataList[1],"=")[1])
	unencodedMessage, _ := url.QueryUnescape(message)
	
	mss.room[roomName].post(p.ActiveSession.GetFullName(),unencodedMessage)
	
	Logger.Debug.Println("User:"+p.ActiveSession.GetFullName()+" from Room: "+roomName+"  <<"+unencodedMessage+">>")
	
	w.Write([]byte(mss.room[roomName].getDiscussion(p.ActiveSession.GetFullName())))
	return "ok", nil
}