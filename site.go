package website

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/jarrancarr/website/html"
)

var ResourceDir = "../../"

type Site struct {
	Name          string
	Session       map[string]*Session
	SessionCookie string
	Tables        *html.TableIndex
	Menus         *html.MenuIndex
	Pages         *PageIndex
}

type Account struct {
	name string
}

type Session struct {
	user *Account
	item map[string]interface{}
}

func CreateSite(name string) *Site {
	site := Site{name, make(map[string]*Session), "", &html.TableIndex{nil}, &html.MenuIndex{nil}, nil}
	return &site
}

func (site *Site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//err := site.home.tmpl.Execute(w, site.home)
	err := site.Pages.Pi["home"].tmpl.Execute(w, site.Pages.Pi["home"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (site *Site) AddMenu(name string) *html.HTMLMenu {
	if site.Menus == nil {
		site.Menus = &html.MenuIndex{nil}
	}
	site.Menus.AddMenu(name)
	return site.Menus.Mi[name]
}

func (site *Site) AddPage(name string, data *Page) *Page {
	if site.Pages == nil {
		site.Pages = &PageIndex{nil}
	}
	site.Pages.AddPage(name, data)
	return site.Pages.Pi[name]
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

func (site *Site) GetSession(key string) *Session {
	return site.Session[key]
}

func (site *Site) CreateSession(user *Account) string {
	key := generateSessionKey()
	site.Session[key] = &Session{user, make(map[string]interface{})}
	return key
}

func generateSessionKey() string {
	return "test"
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
