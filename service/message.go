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
	"strconv"
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
	member	[]*website.Session
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
func (mss *MessageService) Execute(data []string, s *website.Session ,p *website.Page) string {
	mss.lock.Lock()
	Logger.Trace.Println(data[0]+" "+data[1]);
	switch data[0] {
		case "roomList": return mss.roomList(s)
		case "addRoom": mss.createRoom(data[1], "", s)
			mss.lock.Unlock()
			return "ok"
		case "post": 
			if (mss.room[data[1]] != nil) {
				mss.room[data[1]].post(data[2], data[3])
			}
			mss.lock.Unlock()
			return "ok"
		case "exitRoom": mss.exitRoom(mss.room[data[1]],s)
			mss.lock.Unlock()
			return "ok"
	}
	mss.lock.Unlock()
	return "huh?"
}
func (mss *MessageService) Status() string {
	return "good"
}
func (mss *MessageService) createRoom(name, passCode string, s *website.Session) {
	Logger.Trace.Println("mss.createRoom("+name+","+passCode+")");
	if (mss.room[name]==nil) {
		newRoom := Room{passCode, make([]*Message,0), make([]*website.Session,0), sync.Mutex{}}
		mss.room[name] = &newRoom
		mss.join(&newRoom, s)
	} else {
		mss.join(mss.room[name], s)
	}
}
func (mss *MessageService) join(r *Room, s *website.Session) {
	r.member = append(r.member, s)
}
func (mss *MessageService) exitRoom(r *Room, s *website.Session) {
	for idx, asdf := range(r.member) {
		if asdf == s {
			r.member = append(r.member[:idx], r.member[idx+1:]...)
		}
	}
}
func (room *Room) post(author, message string) {
	room.lock.Lock()
	m := Message{message, author, time.Now(), false, nil}
	room.message = append(room.message, &m)
	room.lock.Unlock()
}
func (room *Room) getDiscussion(userName string) string {
	Logger.Trace.Println("room.getDiscussion("+userName+")")
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
func (room *Room) WhoseThere() string {	
	Logger.Debug.Println("room.WhoseThere()")
	room.lock.Lock()
	who := "["
	first := true
	for _, m := range(room.member) {
		if !first {
			who += ","
		} else {
			first = false
		}
		who += `["`+m.GetFullName()+`","`+m.GetUserName()+`"]`
	}
	who += "]"
	room.lock.Unlock()
	Logger.Debug.Println(who)
	return who
}
func (mss *MessageService) roomList(s *website.Session) string {
	Logger.Trace.Println("mss.roomList(<"+s.GetId()+"> *website.Session)")
	roomList := `{ "rooms":{`
	first := true
	for k, r := range(mss.room) {
		if !first {
			roomList += ","
		} else {
			first = false
		}
		roomList += `"`+k+`":`+strconv.Itoa(len(r.member))
	}
	roomList += `}, "conversations":{`
	first = true
	for k, r := range(mss.room) {if !first {
			roomList += ","
		} else {
			first = false
		}
		roomList += `"`+k+`":`+r.getDiscussion(s.GetFullName())
	}
	roomList += `} }`
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
	Logger.Debug.Println("mss.AddRoomAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page)")
	httpData, _ :=ioutil.ReadAll(r.Body)
	if (httpData == nil || len(httpData) == 0) {
		return "", errors.New("No Data")
	}
	dataList := strings.Split(string(httpData),"&")
	roomName := strings.Split(dataList[0],"=")[1]
	roomPass := strings.Split(dataList[1],"=")[1]
	Logger.Debug.Println("new room is "+roomName+" with password:"+roomPass)
	mss.createRoom(roomName,roomPass,s)
	
	w.Write([]byte(mss.roomList(s)))
	return "ok", nil
}
func (mss *MessageService) GetRoomsAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	Logger.Trace.Println("mss.GetRoomsAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page)")
	w.Write([]byte(mss.roomList(s)))
	return "ok", nil
}
func (mss *MessageService) ExitRoomAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	Logger.Trace.Println("mss.ExitRoomAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page)")
	w.Write([]byte(""))
	httpData, _ :=ioutil.ReadAll(r.Body)
	if (httpData == nil || len(httpData) == 0) {
		return "", errors.New("No Data")
	}
	dataList := strings.Split(string(httpData),"&")
	roomName := strings.Split(dataList[0],"=")[1]
	mss.exitRoom(mss.room[roomName], s)
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
func (mss *MessageService) GetRoom(roomName string) (*Room, error) {
	if mss.room[roomName] == nil {
		return nil, errors.New("No room by that name")
	}
	return mss.room[roomName], nil
}