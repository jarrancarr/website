package website

import (
	"bufio"
	"bytes"
	"html/template"
	"net/http"
	"os"
	"strings"
	"strconv"
	"sync"
	//"fmt"

	"github.com/jarrancarr/website/html"
)

type postFunc func(w http.ResponseWriter, r *http.Request, s *Session, p *Page) (string, error)
type filterFunc func(w http.ResponseWriter, r *http.Request, s *Session) (string, error)

type Page struct {
	Title, Url          string
	Body                map[string]map[string][]string 	// page Body Data: map[language][name][Array of string]
	Data                map[string][]template.HTML		// for HTML item arrays
	Script              map[string][]template.JS		// for javascript code arrays
	Site                *Site							// reference to site
	Param			    map[string]string				// temporary parameters
	ParamList		    map[string][]string				// temporary parameters
	ParamMap		    map[string]map[string]string	// temporary parameters
	paramTriggerHandle  map[string]postFunc				// functions executed with URL parameters
	postHandle          map[string]postFunc				// functions executed from a post request
	ajaxHandle          map[string]postFunc				// functions that respond to AJAX requests
	menus               *html.MenuIndex					// menus
	tables              *html.TableIndex				// tables
	tmpl                *template.Template				// this pages HTML template
	pages               *PageIndex						// sub pages
	Parent				*Page							// parent page
	initProcessor       []postFunc 						// initial processors before site processors
	preProcessor        []postFunc 						// processors after site processors
	postProcessor       []postFunc 						// processors after page
	bypassSiteProcessor map[string]bool					// any site processor to not precess for this page
}

type PageIndex struct {
	Pi map[string]*Page
}

var activeSession *Session
var pageLock = &sync.Mutex{}

func LoadPage(site *Site, title, tmplName, url string) (*Page, error) {
	var body map[string]map[string][]string
	if title != "" {
		lang := "en"
		filename := title + ".txt"
		data, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		body = make(map[string]map[string][]string)
		body[lang] = make(map[string][]string)
		r := bufio.NewReader(data)
		s, _, e := r.ReadLine()
		for e == nil {
			if strings.HasPrefix(string(s), "<<lang>>") {
				lang = string(s[8:])
				body[lang] = make(map[string][]string)
			} else {
				field := strings.Split(string(s), ">>")
				items := strings.Split(field[1]," ")
				quotes := false
				stringbuild := ""
				for _, item := range items {
					if quotes {
						stringbuild += " " + item
						if strings.HasSuffix(item, "\"") {
							body[lang][field[0]] = append(body[lang][field[0]],stringbuild[:len(stringbuild)-1])
							quotes = false
						}
					} else if strings.HasPrefix(item, "\"") {
						quotes = true
						stringbuild = item[1:]
					} else {
						body[lang][field[0]] = append(body[lang][field[0]],item)
					}
				}
			}
			s, _, e = r.ReadLine()
		}
	}

	page := &Page{Title:title, Body:body, Site:site, Data:make(map[string][]template.HTML), Script:make(map[string][]template.JS)}
	page.tmpl = template.Must(template.New(tmplName + ".html").Funcs(
		template.FuncMap{
			"table":   		page.table,
			"item":    		page.item,
			"body":    		page.body,
			"service": 		page.service,
			"get": 	   		page.get,
			"page":    		page.page,
			"debug":   		page.debug,
			"menu":    		page.menu,
			"data":    		page.data,
			"param":   		page.getParam,
			"paramList":	page.getParamList,
			"session": 		page.getSessionParam,
			"getList": 		page.getList,
			"ajax":    		page.ajax,
			"target":  		page.target}).
		ParseFiles(ResourceDir + "/templates/" + tmplName + ".html"))
	if url != "" {
		http.HandleFunc(url, page.ServeHTTP)
	}
	return page, nil
}
func (pi *PageIndex) AddPage(name string, data *Page) {
	if pi.Pi == nil {
		pi.Pi = make(map[string]*Page)
	}
	pi.Pi[name] = data
}
func (page *Page) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pageLock.Lock()
	activeSession = page.Site.GetCurrentSession(w, r)
	//fmt.Println("processing page: "+page.Title)
	for _, pFunc := range page.initProcessor {
		status, err := pFunc(w, r, page.Site.GetCurrentSession(w, r), page)
		if status != "ok" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			pageLock.Unlock()
			return
		}
	}
	for key, pFunc := range page.Site.SiteProcessor {
		if page.bypassSiteProcessor == nil || !page.bypassSiteProcessor[key] {
			status, _ := pFunc(w, r, page.Site.GetCurrentSession(w, r), page)
			if status != "ok" {
				http.Redirect(w, r, status, 302)
				pageLock.Unlock()
				return
			}
		}
	}
	for _, pFunc := range page.preProcessor {
		status, err := pFunc(w, r, page.Site.GetCurrentSession(w, r), page)
		if status != "ok" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			pageLock.Unlock()
			return
		}
	}
	paramMap := r.URL.Query() 
	page.Param = make(map[string]string)
	for key, _ := range paramMap {
		page.Param[key] = paramMap.Get(key)
	}
	for key, _ := range paramMap {
		if page.paramTriggerHandle[key] != nil {
			page.paramTriggerHandle[key](w, r, activeSession, page)
		}
		if page.Site.ParamTriggerHandle[key] != nil {
			page.Site.ParamTriggerHandle[key](w, r, activeSession, page)
		}
	}
	
	if r.Method == "POST" {
		//fmt.Println("processing POST: "+r.FormValue("postProcessingHandler"))
		if page.postHandle[r.FormValue("postProcessingHandler")]==nil {
			//fmt.Println("postProcessor is null")
		} else {
			redirect, _ := page.postHandle[r.FormValue("postProcessingHandler")](w, r, activeSession, page)
			if redirect != "" {
				http.Redirect(w, r, redirect, 302)
			} else {
				err := page.tmpl.Execute(w, page)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			}
		}
		pageLock.Unlock()
		return
	} else if r.Method == "AJAX" {
		status, err := page.ajaxHandle[r.Header.Get("ajaxProcessingHandler")](w, r, activeSession, page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		if status != "ok" {
			http.Redirect(w, r, status, 307)
		}
		pageLock.Unlock()
		return
	} else {
		err := page.tmpl.Execute(w, page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
	for _, pFunc := range page.postProcessor {
		status, err := pFunc(w, r, activeSession, page)
		if status != "ok" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			pageLock.Unlock()
			return
		}
	}
	activeSession.Data["navigation"]=r.RequestURI
	pageLock.Unlock()
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
	data.Parent = page
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
func (page *Page) AddParam(name, data string) *Page {
	if (page.Param==nil) {
		page.Param = make(map[string]string)
	}
	page.Param[name] = data
	return page
}
func (page *Page) AddParamList(name string, data []string) *Page {
	if (page.ParamList==nil) {
		page.ParamList = make(map[string][]string)
	}
	page.ParamList[name] = data
	return page
}
func (page *Page) ClearData(name string) {
	page.Data[name] = []template.HTML{}
}
func (page *Page) AddBody(lang, name, line string) *Page {
	page.Body[lang][name] = []string{}
	quotes := false
	stringbuild := ""
	items := strings.Split(line, " ")
	for _, item := range items {
		if quotes {
			stringbuild += " " + item
			if strings.HasSuffix(item, "\"") {
				page.Body[lang][name] = append(page.Body[lang][name],stringbuild[:len(stringbuild)-1])
				quotes = false
			}
		} else if strings.HasPrefix(item, "\"") {
			quotes = true
			stringbuild = item[1:]
		} else {
			page.Body[lang][name] = append(page.Body[lang][name],item)
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
func (page *Page) AddParamTriggerHandler(name string, handle postFunc) *Page {
	if page.paramTriggerHandle == nil {
		page.paramTriggerHandle = make(map[string]postFunc)
	}
	page.paramTriggerHandle[name] = handle
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
				page.Site.Pages.Pi[name[0]].AddParam(pair[0],pair[1])
			}
			return template.HTML(page.Site.Pages.Pi[name[0]].Render())
		}
	}
	for i, d := range(name) {
		if i<1 { continue }
		pair := strings.Split(d,">>")
		page.pages.Pi[name[0]].AddParam(pair[0],pair[1])
	}
	return template.HTML(page.pages.Pi[name[0]].Render())
}
func (page *Page) debug(name ...string) template.HTML {
	all := "<br/><div class='debug'><p><code>page: "+page.Title
	all += "<br/>&nbsp&nbspUrl: "+page.Url
	all += "<br/>&nbsp&nbspBody: "
	for lang, book := range page.Body {
		all += "<br/>&nbsp&nbsp&nbsp&nbspFor language: "+lang
		for key,val := range book {
			all += "<br/>&nbsp&nbsp&nbsp&nbsp&nbsp&nbsp"+key+": "
			for _, w := range val { all += w + " " }
		}
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



func (page *Page) item(name ...string) template.CSS {		// item pulls a string from the parameter text file by name and optionally a 
	return template.CSS(page.body(name...))					// number indicating which index of that string to pull
}
func (page *Page) body(name ...string) string {				// retrieves a body element as a string
	lang := activeSession.GetLang()							// an index parameter will return the Nth in the array
	if page.Body[lang][name[0]] == nil {					// 'param:temp' will populate the index parameter from the Param list
		return ""											// 'language:xx' will get the paramater for language xx
	}
	var item []string
	index := int64(-1)
	var err error
	if len(name) == 1 {
		return page.fullBody(lang, name[0])
	} 
	for _, asdf := range(name[1:]) {
		if strings.HasPrefix(asdf,"param:") {
			index, err = strconv.ParseInt(page.Param[strings.Split(asdf,":")[1]], 10, 64)
		} else if strings.HasPrefix(asdf,"language:") {
			lang = asdf[9:]
		} else {
			index, err = strconv.ParseInt(name[1], 10, 64)
		}
	}
	item = page.Body[lang][name[0]]
	if err != nil {
		return item[0]
	}
	return item[index]
}
func (page *Page) fullBody(lang, name string) string {		// retrieves the entire line of text elements identified by that name
	whole := ""
	for _, s := range page.Body[lang][name] { whole += " "+s }
	return whole[1:]
}
func (page *Page) service(data ...string) template.HTML {	// calls the service by its registered name
	return template.HTML(page.Site.Service[data[0]].Execute(activeSession, data[1:]))
}
func (page *Page) get(data ...string) Item {				// retireves an Item(interface{}) Object
	return page.Site.Service[data[0]].Get(page, activeSession, data[1:])
}
func (page *Page) data(data ...string) template.HTML {		// retireves an HTML data element from the page's Data store
	if page.Data[data[0]] == nil { return "" }
	item := page.Data[data[0]]
	index, err := strconv.ParseInt(data[1], 10, 64)
	if err != nil {
		return template.HTML(item[0])
	}
	return template.HTML(item[index])
}
func (page *Page) getParam(name string) string {			// returns a page's named paramater
	if page.Param==nil || page.Param[name]=="" {
		return ""
	}
	return page.Param[name]
}
func (page *Page) getParamList(name string) []string {			// returns a page's named paramater
	if page.ParamList==nil || page.ParamList[name]==nil {
		return nil
	}
	return page.ParamList[name]
}
func (page *Page) getSessionParam(name string) string {		// returns a session paramater
	if name=="language" {
		return activeSession.GetLang()
	}
	return activeSession.Data[name]
}
func (page *Page) getList(name string) []string {			// returns a pages Body list via a page's paramater name
	lang := activeSession.GetLang()
	return page.Body[lang][page.Param[name]]
}
func (page *Page) ajax(data ...string) template.HTML {		// sets up an ajax call to retrieve data from the server.
	url := page.Url											// this call should be accompanied by a target on the page
	handler := ""											// and the AJAX Handler function
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
