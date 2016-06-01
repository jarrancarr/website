package website

import (
	"bufio"
	"bytes"
	//"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/jarrancarr/website/html"
)

type postFunc func(w http.ResponseWriter, r *http.Request)

type Page struct {
	Title      string
	Body       map[string]string
	Site       *Site
	postHandle map[string]postFunc
	secure     bool
	menus      *html.MenuIndex
	tables     *html.TableIndex
	tmpl       *template.Template
	pages      *PageIndex
}

type PageIndex struct {
	Pi map[string]*Page
}

func (pi *PageIndex) AddPage(name string, data *Page) {
	if pi.Pi == nil {
		pi.Pi = make(map[string]*Page)
	}
	pi.Pi[name] = data
}

func (page *Page) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if page.secure {
		sessionCookie, err := r.Cookie(page.Site.SessionCookie)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if sessionCookie == nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
	}
	if r.Method == "POST" {
		page.postHandle[r.FormValue("postProcessingHandler")](w, r)
		w.Write([]byte("thank you"))
	} else {
		err := page.tmpl.Execute(w, page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func LoadPage(site *Site, title, tmplName, url string) (*Page, error) {
	var body map[string]string
	if title != "" {
		filename := title + ".txt"
		data, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		body = make(map[string]string)
		r := bufio.NewReader(data)
		s, _, e := r.ReadLine()
		for e == nil {
			field := strings.Split(string(s), ">>")
			body[field[0]] = field[1]
			s, _, e = r.ReadLine()
		}
	}

	page := &Page{title, body, site, nil, false, nil, nil, nil, nil}
	page.tmpl = template.Must(template.New(tmplName + ".html").Funcs(
		template.FuncMap{
			"table":   page.table,
			"item":    page.item,
			"service": page.service,
			"page":    page.page,
			"menu":    page.menu}).
		ParseFiles(ResourceDir + "/templates/" + tmplName + ".html"))
	if url != "" {
		http.HandleFunc(url, page.ServeHTTP)
	}
	return page, nil
}

func (page *Page) AddMenu(name string) *html.HTMLMenu {
	if page.menus == nil {
		page.menus = &html.MenuIndex{nil}
	}
	page.menus.AddMenu(name)
	return page.menus.Mi[name]
}

func (page *Page) AddTable(name string, headers, data []string) *html.HTMLTable {
	if page.tables == nil {
		page.tables = &html.TableIndex{nil}
	}
	page.tables.AddTable(name, headers, data)
	return page.tables.Ti[name]
}

func (page *Page) AddPage(name string, data *Page) {
	if page.pages == nil {
		page.pages = &PageIndex{nil}
	}
	page.pages.AddPage(name, data)
}

func (page *Page) AddPostHandler(name string, handle postFunc) {
	if page.postHandle == nil {
		page.postHandle = make(map[string]postFunc)
	}
	page.postHandle[name] = handle
}

func (page *Page) SetSecure() {
	page.secure = true
}

func (page *Page) table(name string) template.HTML {
	if page.tables.Ti[name] == nil {
		return template.HTML(page.Site.Tables.Ti[name].Render())
	}
	return template.HTML(page.tables.Ti[name].Render())
}

func (page *Page) page(name string) template.HTML {
	if page.pages == nil || page.pages.Pi == nil || page.pages.Pi[name] == nil {
		if page.Site.Pages == nil || page.Site.Pages.Pi == nil || page.Site.Pages.Pi[name] == nil {
			return template.HTML("<h1>Empty page</h1>")
		} else {
			return template.HTML(page.Site.Pages.Pi[name].Render())
		}
	}
	return template.HTML(page.pages.Pi[name].Render())
}

func (page *Page) menu(name string) template.HTML {
	if page.menus == nil || page.menus.Mi == nil || page.menus.Mi[name] == nil {
		if page.Site.Menus == nil || page.Site.Menus.Mi[name] == nil {
			return ""
		}
		return template.HTML(page.Site.Menus.Mi[name].Render())
	}
	return template.HTML(page.menus.Mi[name].Render())
}

func (page *Page) item(name string) template.CSS {
	buf := new(bytes.Buffer)
	t := template.Must(template.New("").Parse(page.Body[name]))
	t.Execute(buf, nil)
	return template.CSS(buf.String())
}

func (page *Page) service(name, user, command, data string) template.HTML {
	return template.HTML("")
}

func (page *Page) Render() template.HTML {
	buf := new(bytes.Buffer)
	page.tmpl.Execute(buf, page)
	return template.HTML(buf.String())
}
