package website

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/jarrancarr/website/html"
	"github.com/jarrancarr/website/service"
)

var ResourceDir = "../../"

type Site struct {
	Name         string
	Tables       *html.TableIndex
	Menus        *html.MenuIndex
	Pages        *PageIndex
	Service      map[string]service.Service
	preProcessor []postFunc
}

func CreateSite(name string) *Site {
	site := Site{name, &html.TableIndex{nil}, &html.MenuIndex{nil}, nil, nil, nil}
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

func (site *Site) AddService(name string, serve service.Service) {
	if site.Service == nil {
		site.Service = make(map[string]service.Service)
	}
	site.Service[name] = serve
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
