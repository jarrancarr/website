package website

import (
	"bufio"
	"bytes"
	"html/template"
	"net/http"
	"os"
	"strings"
	"strconv"
	"fmt"

	"github.com/jarrancarr/website/html"
)

type postFunc func(w http.ResponseWriter, r *http.Request, s *Session, p *Page) (string, error)
type filterFunc func(w http.ResponseWriter, r *http.Request, s *Session) (string, error)

type Page struct {
	Title, Url         string
	Body               map[string][]string
	Data               map[string][]template.HTML
	Script             map[string][]template.JS
	Site               *Site
	Param			   map[string]string
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
	fmt.Println("processing page: "+page.Title)
	for _, pFunc := range page.initProcessor {
		status, err := pFunc(w, r, page.Site.GetCurrentSession(w, r), page)
		if status != "ok" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	for key, pFunc := range page.Site.SiteProcessor {
		if page.bypassSiteProcessor == nil || !page.bypassSiteProcessor[key] {
			status, _ := pFunc(w, r, page.Site.GetCurrentSession(w, r), page)
			if status != "ok" {
				http.Redirect(w, r, status, 302)
				return
			}
		}
	}
	for _, pFunc := range page.preProcessor {
		status, err := pFunc(w, r, page.Site.GetCurrentSession(w, r), page)
		if status != "ok" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	paramMap := r.URL.Query() 
	page.Param = make(map[string]string)
	for key, _ := range paramMap {
		page.Param[key] = paramMap.Get(key)
	}
	if r.Method == "POST" {
		//fmt.Println("processing POST: "+r.FormValue("postProcessingHandler"))
		if page.postHandle[r.FormValue("postProcessingHandler")]==nil {
			//fmt.Println("postProcessor is null")
		} else {
			redirect, _ := page.postHandle[r.FormValue("postProcessingHandler")](w, r, page.Site.GetCurrentSession(w, r), page)
			if redirect != "" {
				http.Redirect(w, r, redirect, 302)
			} else {
				activeSession = page.Site.GetCurrentSession(w, r)
				err := page.tmpl.Execute(w, page)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}
		}
		return
	} else if r.Method == "AJAX" {
		status, err := page.ajaxHandle[r.Header.Get("ajaxProcessingHandler")](w, r, page.Site.GetCurrentSession(w, r), page)
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
		status, err := pFunc(w, r, page.Site.GetCurrentSession(w, r), page)
		if status != "ok" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func LoadPage(site *Site, title, tmplName, url string) (*Page, error) {
	var body map[string][]string
	if title != "" {
		filename := title + ".txt"
		data, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		body = make(map[string][]string)
		r := bufio.NewReader(data)
		s, _, e := r.ReadLine()
		for e == nil {
			field := strings.Split(string(s), ">>")
			items := strings.Split(field[1]," ")
			quotes := false
			stringbuild := ""
			for _, item := range items {
				if quotes {
					stringbuild += " " + item
					if strings.HasSuffix(item, "\"") {
						body[field[0]] = append(body[field[0]],stringbuild[:len(stringbuild)-1])
						quotes = false
					}
				} else if strings.HasPrefix(item, "\"") {
					quotes = true
					stringbuild = item[1:]
				} else {
					body[field[0]] = append(body[field[0]],item)
				}
			}
			s, _, e = r.ReadLine()
		}
	}

	page := &Page{Title:title, Body:body, Site:site, Data:make(map[string][]template.HTML), Script:make(map[string][]template.JS)}
	page.tmpl = template.Must(template.New(tmplName + ".html").Funcs(
		template.FuncMap{
			"table":   page.table,
			"item":    page.item,
			"body":    page.body,
			"service": page.service,
			"page":    page.page,
			"debug":    page.debug,
			"menu":    page.menu,
			"data":    page.data,
			"param":   page.getParam,
			"getParamList":   page.getParamList,
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

func (page *Page) AddScript(name, script string) *Page {
	page.Script[name] = append(page.Script[name], template.JS(script))
	return page
}

func (page *Page) AddData(name, data string) *Page {
	page.Data[name] = append(page.Data[name], template.HTML(data))
	return page
}

func (page *Page) ClearData(name string) {
	page.Data[name] = []template.HTML{}
}

func (page *Page) AddBody(name, line string) *Page {
	page.Body[name] = []string{}
	quotes := false
	stringbuild := ""
	items := strings.Split(line, " ")
	for _, item := range items {
		if quotes {
			stringbuild += " " + item
			if strings.HasSuffix(item, "\"") {
				page.Body[name] = append(page.Body[name],stringbuild[:len(stringbuild)-1])
				quotes = false
			}
		} else if strings.HasPrefix(item, "\"") {
			quotes = true
			stringbuild = item[1:]
		} else {
			page.Body[name] = append(page.Body[name],item)
		}
	}
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

func (page *Page) page(name ...string) template.HTML {
	if page.pages == nil || page.pages.Pi == nil || page.pages.Pi[name[0]] == nil {
		if page.Site.Pages == nil || page.Site.Pages.Pi == nil || page.Site.Pages.Pi[name[0]] == nil {
			return template.HTML("<h1>Empty page</h1>")
		} else {
			for i, d := range(name) {
				if i<1 { continue }
				pair := strings.Split(d,">>")
				page.Site.Pages.Pi[name[0]].AddBody(pair[0],pair[1])
			}
			return template.HTML(page.Site.Pages.Pi[name[0]].Render())
		}
	}
	for i, d := range(name) {
		if i<1 { continue }
		pair := strings.Split(d,">>")
		page.pages.Pi[name[0]].AddBody(pair[0],pair[1])
	}
	return template.HTML(page.pages.Pi[name[0]].Render())
}

func (page *Page) debug(name ...string) template.HTML {
	all := "<br/><div class='debug'><p><code>page: "+page.Title
	all += "<br/>&nbsp&nbspUrl: "+page.Url
	all += "<br/>&nbsp&nbspBody: "
	for key,val := range page.Body {
		all += "<br/>&nbsp&nbsp&nbsp&nbsp"+key+": "
		for _, w := range val { all += w + " " }
	}
	all += "<br/>&nbsp&nbspData: "
	for key,val := range page.Data {
		all += "<br/>&nbsp&nbsp&nbsp&nbsp"+key+": "
		for _, w := range val { all += string(w) + " " }
	}
	all += "<br/>&nbsp&nbspScript: "
	for key,val := range page.Script {
		all += "<br/>&nbsp&nbsp&nbsp&nbsp"+key+": "
		for _, w := range val { 
			all += "<br/>&nbsp&nbsp&nbsp&nbsp&nbsp&nbsp"+string(w) 
		}
	}
	all += "<br/>&nbsp&nbspparam: "
	for key,val := range page.Param {
		all += "<br/>&nbsp&nbsp&nbsp&nbsp"+key+": "+val
	}
	all += "<br/>&nbsp&nbsppostHandle: "
	for key,_ := range page.postHandle {
		all += "<br/>&nbsp&nbsp&nbsp&nbsp"+key
	}
	all += "<br/>&nbsp&nbspajaxHandle: "
	for key,_ := range page.ajaxHandle {
		all += "<br/>&nbsp&nbsp&nbsp&nbsp"+key
	}
	all += "</code></p></div>"
	return template.HTML(all)
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

// item pulls a string from the parameter text file by name and optionally a 
// number indicating which index of that string to pull
func (page *Page) item(name ...string) template.CSS {
	return template.CSS(page.body(name...))
}

func (page *Page) body(name ...string) string {
	if page.Body[name[0]] == nil {
		return "" //page.Site.item(name)
	}
	var item []string
	var index int64
	var err error
	if len(name) == 1 {
		return page.fullBody(name[0])
	} 
	item = page.Body[name[0]]
	if strings.HasPrefix(name[1],"Body:") {
		index, err = strconv.ParseInt(page.Body[strings.Split(name[1],":")[1]][0], 10, 64)
	} else {
		index, err = strconv.ParseInt(name[1], 10, 64)
	}
	if err != nil {
		return item[0]
	}
	return item[index]
}

func (page *Page) fullBody(name string) string {
	whole := ""
	for _, s := range page.Body[name] { whole += " "+s }
	return whole[1:]
}

func (page *Page) service(data ...string) template.HTML {
	return template.HTML(page.Site.Service[data[0]].Execute(activeSession, data[1:]))
}

func (page *Page) data(data ...string) template.HTML {
	if page.Data[data[0]] == nil { return "" }
	item := page.Data[data[0]]
	index, err := strconv.ParseInt(data[1], 10, 64)
	if err != nil {
		return template.HTML(item[0])
	}
	return template.HTML(item[index])
}

func (page *Page) getParam(name string) string {
	if page.Param==nil || page.Param[name]=="" {
		return ""
	}
	return page.Param[name]
}

func (page *Page) getParamList(name string) []string {
	return page.Body[page.Param[name]]
}

func (page *Page) ajax(data ...string) template.HTML {
	url := page.Url
	handler := ""
	trigger := ""
	target := ""
	onClick := ""
	item := "$(document.createElement('li')).text( i + ' - ' + val )"
	jsData := "'greet':'hello there, partner!'"
	variables := ""
	success := ""
	for _, d := range(data) {
		if strings.HasPrefix(d, "url:") { url = d[4:] }
		if strings.HasPrefix(d, "handler:") { handler = d[8:] }
		if strings.HasPrefix(d, "target:") { target = d[7:] }
		if strings.HasPrefix(d, "trigger:") { trigger = d[8:] }
		if strings.HasPrefix(d, "data:") { jsData = d[5:] }
		if strings.HasPrefix(d, "item:") { item = d[5:] }
		if strings.HasPrefix(d, "onclick:") { onClick = d[8:] }
		if strings.HasPrefix(d, "var:") { variables += "var " + d[4:] + "; " }
		if strings.HasPrefix(d, "success:") { success = d[8:] }
	}
	if success == "" {
		success = `var ul = $( "<ul/>", {"class": "my-new-list"});
			var obj = JSON.parse(data);	$("#`+target+`").empty(); $("#`+target+`").append(ul);
			$.each(obj, function(i,val) { item =`+item+`; `+onClick+` ul.append( item ); });`
	}
	return template.HTML(`<script>`+variables+`
		$(function() {
			$('#`+trigger+`-trigger').on('click', function() {
				$.ajax({
					url: '/`+url+`',
					type: 'AJAX',
					headers: { 'ajaxProcessingHandler':'`+handler+`' },
					dataType: 'html',
					data: { `+jsData+` },
					success: function(data, textStatus, jqXHR) {
						`+success+`	
					},
					error: function(data, textStatus, jqXHR) {
						console.log("button fail!");
					}
				});
			});
		});
	</script>`)
}

//sets up a div target for the ajax call
func (page *Page) target(name string) template.HTML {
	return template.HTML("<div id='"+name+"'></div>")
}

func (page *Page) Render() template.HTML {
	buf := new(bytes.Buffer)
	page.tmpl.Execute(buf, page)
	return template.HTML(buf.String())
}
