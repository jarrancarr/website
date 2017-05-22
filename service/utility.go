package service

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jarrancarr/website"
)

type UtilityService struct {
	hook []website.PostFunc
	acs  *website.AccountService
	lock sync.Mutex
}

func CreateUtilityService(acs *website.AccountService) *UtilityService {
	Logger.Trace.Println()
	uts := UtilityService{nil, acs, sync.Mutex{}}
	return &uts
}
func (uts *UtilityService) Execute(data []string, s *website.Session, p *website.Page) string {
	uts.lock.Lock()
	Logger.Debug.Println("UtilityService.Execute(" + data[0] + ", page<>)")
	switch data[0] {
	case "date":
		uts.lock.Unlock()
		days, err := strconv.Atoi(data[1])
		if err == nil {
			return time.Now().AddDate(0, 0, days).Format("Monday")
		}
		return time.Now().Format("Monday")
	}
	uts.lock.Unlock()
	return "No process for commands: " + strings.Join(data, "|")
}
func (uts *UtilityService) Status() string {
	return "good"
}
func (uts *UtilityService) Metrics(what ...string) int {
	switch what[0] {
	}
	return 0
}
func (uts *UtilityService) Get(p *website.Page, s *website.Session, data []string) website.Item {
	Logger.Trace.Println("UtilityService.Get(page<" + p.Title + ">, session<" + s.GetUserName() + ">, " + strings.Join(data, "|") + ")")

	switch data[0] {
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
