package website

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jarrancarr/website/html"
)

var ResourceDir = "../../"

type Site struct {
	Name                  string
	SiteSessionCookieName string
	Tables                *html.TableIndex
	Menus                 *html.MenuIndex
	Pages                 *PageIndex
	UserSession           map[string]*Session
	Service               map[string]Service
	SiteProcessor         map[string]postFunc
}

type Session struct {
	Item map[string]interface{}
	Data map[string]string
}

func createSession() *Session {
	return &Session{make(map[string]interface{}), make(map[string]string)}
}

func CreateSite(name string) *Site {
	site := Site{name, name + "-cookie", &html.TableIndex{nil}, &html.MenuIndex{nil}, nil, make(map[string]*Session), nil, nil}
	return &site
}

func (site *Site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//err := site.home.tmpl.Execute(w, site.home)
	err := site.Pages.Pi["home"].tmpl.Execute(w, site.Pages.Pi["home"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (site *Site) GetCurrentSession(w http.ResponseWriter, r *http.Request) *Session {
	sessionCookie, _ := r.Cookie(site.SiteSessionCookieName)
	//	cookies := r.Cookies()
	//	fmt.Printf("found %v cookies\n", len(cookies))
	//	for _, c := range cookies {
	//		fmt.Println(c.Name + " = " + c.Value)
	//	}
	if sessionCookie != nil && site.UserSession[sessionCookie.Value] != nil {
		fmt.Println("sessionCookie: " + sessionCookie.Name + " = " + sessionCookie.Value)
		return site.UserSession[sessionCookie.Value]
	} else {
		sessionKey := make([]byte, 64)
		rand.Read(sessionKey)
		fmt.Println("creating cookie: " + base64.URLEncoding.EncodeToString(sessionKey))
		http.SetCookie(w, &http.Cookie{site.SiteSessionCookieName, base64.URLEncoding.EncodeToString(sessionKey), "/",
			"localhost", time.Now().Add(time.Hour * 24), "", 50000, false, true, "none=none", []string{"none=none"}})
		site.UserSession[base64.URLEncoding.EncodeToString(sessionKey)] = createSession()
		site.UserSession[base64.URLEncoding.EncodeToString(sessionKey)].Data["name"] = "Anonamous"

		return site.UserSession[base64.URLEncoding.EncodeToString(sessionKey)]
	}
	return nil
}

func (site *Site) AddSiteProcessor(name string, initFunc postFunc) {
	if site.SiteProcessor == nil {
		site.SiteProcessor = make(map[string]postFunc)
	}
	site.SiteProcessor[name] = initFunc
}

func (site *Site) AddMenu(name string) *html.HTMLMenu {
	if site.Menus == nil {
		site.Menus = &html.MenuIndex{nil}
	}
	site.Menus.AddMenu(name)
	return site.Menus.Mi[name]
}

func (site *Site) AddPage(name, template, url string) *Page {
	page, err := LoadPage(site, name, template, url)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if site.Pages == nil {
		site.Pages = &PageIndex{nil}
	}
	site.Pages.AddPage(name, page)
	return page
}

func (site *Site) AddService(name string, serve Service) Service {
	if site.Service == nil {
		site.Service = make(map[string]Service)
	}
	site.Service[name] = serve
	return serve
}

func (site *Site) upload(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	file, handler, err := r.FormFile("uploadfile")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()
	f, err := os.OpenFile("../../temp/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	io.Copy(f, file)
	site.ServeHTTP(w, r)
}

func ServeResource(w http.ResponseWriter, r *http.Request) {
	path := ResourceDir + "/public" + r.URL.Path
	if strings.HasSuffix(r.URL.Path, "js") {
		w.Header().Add("Content-Type", "application/javascript")
	} else if strings.HasSuffix(r.URL.Path, "css") {
		w.Header().Add("Content-Type", "text/css")
	} else if strings.HasSuffix(r.URL.Path, "png") {
		w.Header().Add("Content-Type", "image/svg+xml")
	} else if strings.HasSuffix(r.URL.Path, "svg") {
		w.Header().Add("Content-Type", "image/svg+xml")
	}

	data, err := ioutil.ReadFile(path)

	if err == nil {
		w.Write(data)
	} else {
		w.WriteHeader(404)
		w.Write([]byte("404, My Friend - " + http.StatusText(404)))
	}
}
