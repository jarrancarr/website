package service

import (
	"net/http"
	"net/url"
	"time"

	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"

	"github.com/jarrancarr/website"
)

var (
	Logger *website.Log
)

type Message struct {
	message, author string
	timestamp       time.Time
	read            bool
	reference       *Message
}
type Room struct {
	passCode string
	message  []*Message
	member   []*website.Session
	lock     sync.Mutex
}
type MessageService struct {
	room map[string]*Room
	hook []website.PostFunc
	acs  *website.AccountService
	lock sync.Mutex
}
type PersonalMessageQueue struct {
	messages []*Message
	lastRead int
	lastPost int
}

func CreateMessageService(acs *website.AccountService) *MessageService {
	Logger.Trace.Println()
	mss := MessageService{make(map[string]*Room), nil, acs, sync.Mutex{}}
	return &mss
}
func (mss *MessageService) Execute(data []string, s *website.Session, p *website.Page) string {
	mss.lock.Lock()
	Logger.Trace.Println("MessageService.Execute(" + data[0] + ", page<>)")
	switch data[0] {
	case "roomList":
		return mss.roomList(s)
	case "addRoom":
		mss.createRoom(data[1], "", s)
		mss.lock.Unlock()
		return "ok"
	case "post":
		if mss.room[data[1]] != nil {
			mss.room[data[1]].post(data[2], data[3])
		}
		mss.lock.Unlock()
		return "ok"
	case "exitRoom":
		mss.exitRoom(mss.room[data[1]], s)
		mss.lock.Unlock()
		return "ok"
	case "#activeRooms":
		mss.lock.Unlock()
		return fmt.Sprintf("%d", len(mss.room))
	}
	mss.lock.Unlock()
	return "huh?"
}
func (mss *MessageService) Status() string {
	return "good"
}
func (mss *MessageService) Metrics(what ...string) int {
	switch what[0] {
	case "rooms":
		return len(mss.room)
	case "totalMessages":
		return 0
	case "room":
		switch what[2] {
		case "messages":
			return len(mss.room[what[1]].message)
		case "members":
			return len(mss.room[what[1]].member)
		}
	}
	return 0
}
func (mss *MessageService) createRoom(name, passCode string, s *website.Session) {
	Logger.Trace.Println("mss.createRoom(" + name + "," + passCode + ")")
	if mss.room[name] == nil {
		newRoom := Room{passCode, make([]*Message, 0), make([]*website.Session, 0), sync.Mutex{}}
		mss.room[name] = &newRoom
		mss.join(&newRoom, s)
	} else {
		mss.join(mss.room[name], s)
	}
}
func (mss *MessageService) AddHook(h website.PostFunc) {
	if mss.hook == nil {
		mss.hook = make([]website.PostFunc, 1)
	}
	mss.hook = append(mss.hook, h)
}
func (mss *MessageService) join(r *Room, s *website.Session) {
	r.member = append(r.member, s)
}
func (mss *MessageService) exitRoom(r *Room, s *website.Session) {
	if r == nil {
		return
	}
	for idx, asdf := range r.member {
		if asdf == s {
			r.member = append(r.member[:idx], r.member[idx+1:]...)
		}
	}
}
func (room *Room) post(author, message string) {
	Logger.Trace.Println("MessageService.post(" + author + ", " + message + ")")
	room.lock.Lock()
	m := Message{message, author, time.Now(), false, nil}
	room.message = append(room.message, &m)
	if time.Since(room.message[0].timestamp) > time.Minute {
		room.message = room.message[1:]
	}
	room.lock.Unlock()
}
func (room *Room) getDiscussion(userName string) string {
	Logger.Trace.Println("room.getDiscussion(" + userName + ")")
	room.lock.Lock()
	discussion := "["
	first := true
	for _, m := range room.message {
		if !first {
			discussion += ","
		} else {
			first = false
		}
		if m.author == userName {
			discussion += `{"author":"","message":"` + url.QueryEscape(m.message) + `"}`
		} else {
			discussion += `{"author":"` + m.author + `","message":"` + url.QueryEscape(m.message) + `"}`
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
	for _, m := range room.member {
		if !first {
			who += ","
		} else {
			first = false
		}
		who += `["` + m.GetFullName() + `","` + m.GetUserName() + `"]`
	}
	who += "]"
	room.lock.Unlock()
	return who
}
func (mss *MessageService) roomList(s *website.Session) string {
	Logger.Trace.Println("mss.roomList(<" + s.GetId() + "> *website.Session)")
	roomList := `{ "rooms":{`
	first := true
	for k, r := range mss.room {
		if !first {
			roomList += ","
		} else {
			first = false
		}
		roomList += `"` + k + `":` + strconv.Itoa(len(r.member))
	}
	roomList += `}, "conversations":{`
	first = true
	for k, r := range mss.room {
		if !first {
			roomList += ","
		} else {
			first = false
		}
		roomList += `"` + k + `":` + r.getDiscussion(s.GetFullName())
	}
	roomList += `} }`
	return roomList
}
func (mss *MessageService) conversations(s *website.Session, roomlist []string) string {
	Logger.Trace.Println("mss.conversations(<" + s.GetId() + "> *website.Session, [" + strings.Join(roomlist, ",") + "])")
	roomList := `{ "conversations":{`
	first := true
	for _, room := range roomlist {
		if mss.room[room] != nil {
			if !first {
				roomList += ","
			} else {
				first = false
			}
			roomList += `"` + room + `":` + mss.room[room].getDiscussion(s.GetFullName())
		}
	}
	roomList += `,"family":` + mss.room[s.GetData("userName")].getDiscussion(s.GetFullName()) + `} `
	roomList += `,"notifications":` + s.GetData("notifications") + `} `
	s.AddData("notifications", "[]")

	return roomList
}
func (mss *MessageService) Get(p *website.Page, s *website.Session, data []string) website.Item {
	Logger.Trace.Println("MessageService.Get(page<" + p.Title + ">, session<" + s.GetUserName() + ">, " + strings.Join(data, "|") + ")")

	switch data[0] {
	case "getAllRooms":
		var answ []interface{}
		mss.lock.Lock()
		for name, room := range mss.room {
			answ = append(answ, struct {
				Name                string
				Messages, Occupance int
			}{
				name,
				len(room.message),
				len(room.member),
			})
		}
		mss.lock.Unlock()
		return answ
	case "occupance":
		var answ []interface{}
		mss.room[data[1]].lock.Lock()
		for _, occupant := range mss.room[data[1]].member {
			answ = append(answ, struct {
				Name, UserName string
			}{
				occupant.Data["name"],
				occupant.Data["userName"],
			})
		}
		mss.room[data[1]].lock.Unlock()
		return answ
	case "getMessages":
		var answ []interface{}
		mss.room[data[1]].lock.Lock()
		for _, message := range mss.room[data[1]].message {
			answ = append(answ, struct {
				TimeStamp, Author, Message string
			}{
				message.timestamp.Format("15:04:05.000"),
				message.author,
				message.message,
			})
		}
		mss.room[data[1]].lock.Unlock()
		return answ
	}

	t := "Duke"
	n := "Bingo"
	d := "The Man!"
	return struct {
		Title, Name, Desc string
	}{
		t, n, d,
	}
}
func (mss *MessageService) CreateRoomAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	Logger.Debug.Println("CreateRoomAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page)")
	_, httpData := website.PullData(r)
	Logger.Debug.Println("new room is " + httpData["roomName"] + " with password:" + httpData["roomPass"])
	mss.createRoom(httpData["roomName"], httpData["roomPass"], s)
	w.Write([]byte(mss.roomList(s)))
	return mss.processHooks(w, r, s, p)
}
func (mss *MessageService) GetRoomsAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	Logger.Trace.Println("mss.GetRoomsAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page)")
	w.Write([]byte(mss.roomList(s)))
	return "ok", nil
}
func (mss *MessageService) GetConversationsAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	Logger.Trace.Println("mss.GetConversationsAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page)")
	obj, _ := website.PullData(r)
	s.Item["config"] = obj["config"]
	if obj["rooms"] == nil {
		return mss.processHooks(w, r, s, p)
	}
	room := obj["rooms"].([]interface{})
	list := make([]string, len(room))
	for i := 0; i < len(room); i++ {
		list[i] = room[i].(string)
	}
	list = append(list, "family")
	conv := mss.conversations(s, list)
	w.Write([]byte(conv))
	return mss.processHooks(w, r, s, p)
}
func (mss *MessageService) ExitRoomAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	Logger.Trace.Println("mss.ExitRoomAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page)")
	w.Write([]byte(""))
	httpData, _ := ioutil.ReadAll(r.Body)
	if httpData == nil || len(httpData) == 0 {
		return "", errors.New("No Data")
	}
	dataList := strings.Split(string(httpData), "&")
	roomName := strings.Split(dataList[0], "=")[1]
	mss.exitRoom(mss.room[roomName], s)
	return "ok", nil
}
func (mss *MessageService) MessageAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	Logger.Trace.Println("mss.MessageAJAXHandler(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page)")
	httpData, _ := ioutil.ReadAll(r.Body)
	if httpData == nil || len(httpData) == 0 {
		return "", errors.New("No Data")
	}
	dataList := strings.Split(string(httpData), "&")
	roomName := strings.Split(dataList[0], "=")[1]
	message, _ := url.QueryUnescape(strings.Split(dataList[1], "=")[1])
	unencodedMessage, _ := url.QueryUnescape(message)

	if roomName == "" || mss.room[roomName] == nil {
		return "", errors.New("Invalid room name")
	}
	mss.room[roomName].post(p.ActiveSession.GetFullName(), unencodedMessage)

	w.Write([]byte(mss.room[roomName].getDiscussion(p.ActiveSession.GetFullName())))
	return mss.processHooks(w, r, s, p)
}
func (mss *MessageService) GetRoom(roomName string) (*Room, error) {
	if mss.room[roomName] == nil {
		return nil, errors.New("No room by that name")
	}
	return mss.room[roomName], nil
}
func (mss *MessageService) processHooks(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	if mss.hook == nil {
		return "ok", nil
	}
	for _, pFunc := range mss.hook {
		if pFunc == nil {
			continue
		}
		status, err := pFunc(w, r, s, p)
		if status != "ok" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return "", err
		}
	}
	return "ok", nil
}
