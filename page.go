package website

import (
	"bufio"
	"bytes"
	//"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"strconv"

	"github.com/jarrancarr/website/html"
)

type postFunc func(w http.ResponseWriter, r *http.Request, s *Session) (string, error)

type filterFunc func(w http.ResponseWriter, r *http.Request, s *Session) (string, error)

type Page struct {
	Title              string
	Body               map[string]string
	Data               map[string][]template.HTML
	Script             map[string][]template.JS
	Site               *Site
	postHandle         map[string]postFunc
	ajaxHandle         map[string]postFunc
	menus              *html.MenuIndex
	tables             *html.TableIndex
	tmpl               *template.Template
	pages              *PageIndex
	initProcessor      []postFunc // initial processors before site processors
	preProcessor       []postFunc // processors after site processors
	postProcessor      []postFunc // processors after page
	bypassSiteProcessor map[string]bool
}

type PageIndex struct {
	Pi map[string]*Page
}

var activeSession *Session

func (pi *PageIndex) AddPage(name string, data *Page) {
	if pi.Pi == nil {
		pi.Pi = make(map[string]*Page)
	}
	pi.Pi[name] = data
}

func (page *Page) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("processing page: "+page.Title)
	for _, pFunc := range page.initProcessor {
		status, err := pFunc(w, r, page.Site.GetCurrentSession(w, r))
		if status != "ok" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	for key, pFunc := range page.Site.SiteProcessor {
		if page.bypassSiteProcessor == nil || !page.bypassSiteProcessor[key] {
			status, _ := pFunc(w, r, page.Site.GetCurrentSession(w, r))
			if status != "ok" {
				http.Redirect(w, r, status, 302)
				return
			}
		}
	}
	for _, pFunc := range page.preProcessor {
		status, err := pFunc(w, r, page.Site.GetCurrentSession(w, r))
		if status != "ok" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if r.Method == "POST" {
		redirect, _ := page.postHandle[r.FormValue("postProcessingHandler")](w, r, page.Site.GetCurrentSession(w, r))
		http.Redirect(w, r, redirect, 302)
		return
	} else if r.Method == "AJAX" {
		status, err := page.ajaxHandle[r.Header.Get("ajaxProcessingHandler")](w, r, page.Site.GetCurrentSession(w, r))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		if status != "ok" {
			http.Redirect(w, r, status, 307)
		}
		return
	} else {
		activeSession = page.Site.GetCurrentSession(w, r)
		err := page.tmpl.Execute(w, page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
	for _, pFunc := range page.postProcessor {
		status, err := pFunc(w, r, page.Site.GetCurrentSession(w, r))
		if status != "ok" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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

	page := &Page{Title:title, Body:body, Site:site, Data:make(map[string][]template.HTML), Script:make(map[string][]template.JS)}
	page.tmpl = template.Must(template.New(tmplName + ".html").Funcs(
		template.FuncMap{
			"table":   page.table,
			"item":    page.item,
			"service": page.service,
			"page":    page.page,
			"menu":    page.menu,
			"ajax":    page.ajax,
			"target":  page.target}).
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

func (page *Page) AddPage(name string, data *Page) *Page {
	if page.pages == nil {
		page.pages = &PageIndex{nil}
	}
	page.pages.AddPage(name, data)
	return page
}

func (page *Page) AddPostHandler(name string, handle postFunc) *Page {
	if page.postHandle == nil {
		page.postHandle = make(map[string]postFunc)
	}
	page.postHandle[name] = handle
	return page
}

func (page *Page) AddAJAXHandler(name string, handle postFunc) *Page {
	if page.ajaxHandle == nil {
		page.ajaxHandle = make(map[string]postFunc)
	}
	page.ajaxHandle[name] = handle
	return page
}

func (page *Page) AddInitProcessor(initFunc postFunc) {
	page.initProcessor = append(page.initProcessor, initFunc)
}

func (page *Page) AddPreProcessor(initFunc postFunc) {
	page.preProcessor = append(page.preProcessor, initFunc)
}

func (page *Page) AddPostProcessor(initFunc postFunc) {
	page.postProcessor = append(page.postProcessor, initFunc)
}

func (page *Page) AddBypassSiteProcessor(name string) {
	if 	page.bypassSiteProcessor ==nil {
		page.bypassSiteProcessor = make(map[string]bool)
	}
	page.bypassSiteProcessor[name] = true
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

func (page *Page) item(name ...string) template.CSS {
	if len(name) == 1 {
		return template.CSS(page.Body[name[0]])
	} 
	item := strings.Split(page.Body[name[0]], " ")
	index, err := strconv.ParseInt(name[1], 10, 32)
	if err != nil {
		return template.CSS(item[0])
	}
	return template.CSS(item[index])
}

func (page *Page) service(data ...string) template.HTML {
	return template.HTML(page.Site.Service[data[0]].Execute(activeSession, data))
}

func (page *Page) ajax(data ...string) template.HTML {
	return template.HTML(`<script>
		$(function() {
			$('input').on('click', function() {
				$.ajax({
					url: '/`+data[0]+`',
					type: 'AJAX',
					headers: { 'ajaxProcessingHandler':'`+data[1]+`' },
					dataType: 'html',
					data: { 'greet':'hello there, partner!' },
					success: function(data, textStatus, jqXHR) {
						var ul = $( "<ul/>", {"class": "my-new-list"});
						var obj = JSON.parse(data);
						$("#`+data[2]+`").replaceWith(ul);
						ul.append( $(document.createElement('li')).text( obj.one ) );
						ul.append( $(document.createElement('li')).text( obj.two ) );
						ul.append( $(document.createElement('li')).text( obj.three ) );
						
					},
					error: function(data, textStatus, jqXHR) {
						console.log("button fail!");
					}
				});
			});
		});
	</script>`)
}

func (page *Page) target(name string) template.HTML {
	return template.HTML("<div id='"+name+"'></div>")
}

func (page *Page) Render() template.HTML {
	buf := new(bytes.Buffer)
	page.tmpl.Execute(buf, page)
	return template.HTML(buf.String())
}
