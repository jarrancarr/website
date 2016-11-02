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

type Page struct {
	Title, Url          string
	Data                map[string]map[string][]string 	// page Data: map[language][name][Array of string]
	DataMap		    	map[string]map[string]string	// parameter map
	Html				*html.HTMLStow					// generic html tag snippets
	Site                *Site							// reference to site
	Parent				*Page							// parent page
	pages               *PageStow						// sub pages
	tables              *html.TableStow					// tables
	Param			    map[string]string				// parameters: scope - name
	ParamList		    map[string][]string				// parameter list: scope - name
	paramTriggerHandle  map[string]postFunc				// functions executed with URL parameters
	postHandle          map[string]postFunc				// functions executed from a post request
	ajaxHandle          map[string]postFunc				// functions that respond to AJAX requests
	tmpl                *template.Template				// this pages HTML template
	initProcessor       []postFunc 						// initial processors before site processors
	preProcessor        []postFunc 						// processors after site processors
	postProcessor       []postFunc 						// processors after page
	bypassSiteProcessor map[string]bool					// any site processor to not precess for this page
}

type PageStow struct {
	Ps map[string]*Page
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
			if string(s) != "" {
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
			}
			s, _, e = r.ReadLine()
		}
	}

	page := &Page{	Title:title, Data:body, Site:site, Html:html.MakeStow()}
	page.tmpl = template.Must(template.New(tmplName + ".html").Funcs(
		template.FuncMap{
			"table":   		page.table,
			"item":    		page.item,
			"service": 		page.service,
			"get": 	   		page.get,
			"page":    		page.page,
			"debug":   		page.debug,
			"html":    		page.getHtml,
			"data":    		page.data,
			"param":   		page.getParam,
			"paramList":	page.getParamList,
			"session": 		page.getSessionParam,
			"getList": 		page.getListByParam,
			"ajax":    		page.ajax,
			"target":  		page.target}).
		ParseFiles(ResourceDir + "/templates/" + tmplName + ".html"))
	if url != "" {
		http.HandleFunc(url, page.ServeHTTP)
	}
	return page, nil
}
func (ps *PageStow) AddPage(name string, data *Page) {
	if ps.Ps == nil {
		ps.Ps = make(map[string]*Page)
	}
	ps.Ps[name] = data
}
func (page *Page) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pageLock.Lock()
	activeSession = page.Site.GetCurrentSession(w, r)
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
		if page.postHandle[r.FormValue("postProcessingHandler")]==nil {
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
		// A normal GET request
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
func (page *Page) AddTable(name string, headers, data []string) *html.HTMLTable {
	if page.tables == nil {
		page.tables = &html.TableStow{nil}
	}
	page.tables.AddTable(name, headers, data)
	return page.tables.Ts[name]
}
func (page *Page) AddPage(name string, data *Page) *Page {
	if page.pages == nil {
		page.pages = &PageStow{nil}
	}
	data.Parent = page
	page.pages.AddPage(name, data)
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
func (page *Page) AddData(lang, name, line string) *Page {
	page.Data[lang][name] = []string{}
	quotes := false
	stringbuild := ""
	items := strings.Split(line, " ")
	for _, item := range items {
		if quotes {
			stringbuild += " " + item
			if strings.HasSuffix(item, "\"") {
				page.Data[lang][name] = append(page.Data[lang][name],stringbuild[:len(stringbuild)-1])
				quotes = false
			}
		} else if strings.HasPrefix(item, "\"") {
			quotes = true
			stringbuild = item[1:]
		} else {
			page.Data[lang][name] = append(page.Data[lang][name],item)
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
	if page.tables.Ts[name] == nil {
		return template.HTML(page.Site.Tables.Ts[name].Render())
	}
	return template.HTML(page.tables.Ts[name].Render())
}
func (page *Page) page(name ...string) template.HTML {
	if page.pages == nil || page.pages.Ps == nil || page.pages.Ps[name[0]] == nil {
		if page.Site.Pages == nil || page.Site.Pages.Ps == nil || page.Site.Pages.Ps[name[0]] == nil {
			return template.HTML("<h1>Empty page</h1>")
		} else {
			for i, d := range(name) {
				if i<1 { continue }
				pair := strings.Split(d,">>")
				page.Site.Pages.Ps[name[0]].AddParam(pair[0],pair[1])
			}
			return template.HTML(page.Site.Pages.Ps[name[0]].Render())
		}
	}
	for i, d := range(name) {
		if i<1 { continue }
		pair := strings.Split(d,">>")
		page.pages.Ps[name[0]].AddParam(pair[0],pair[1])
	}
	return template.HTML(page.pages.Ps[name[0]].Render())
}
func (page *Page) debug(name ...string) template.HTML {
	all := "<br/><div class='debug'><p><code>page: "+page.Title
	all += "<br/>&nbsp&nbspUrl: "+page.Url
	all += "<br/>&nbsp&nbspData: "
	for lang, book := range page.Data {
		all += "<br/>&nbsp&nbsp&nbsp&nbspFor language: "+lang
		for key,val := range book {
			all += "<br/>&nbsp&nbsp&nbsp&nbsp&nbsp&nbsp"+key+": "
			for _, w := range val { all += w + " " }
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
func (page *Page) getHtml(name string) template.HTML {
	if page.Html == nil {
		if page.Parent == nil {
			return page.Site.GetHtml(name)
		}
		return page.Parent.getHtml(name)
	}
	return template.HTML(page.Html.Hs[name].Render())
}
func (page *Page) item(name ...string) template.CSS {		// item pulls a string from the parameter text file by name and optionally a 
	return template.CSS(page.data(name...))					// number indicating which index of that string to pull
}
func (page *Page) data(name ...string) string {				// retrieves a data element as a string
	lang := activeSession.GetLang()							// an index parameter will return the Nth in the array
	if page.Data[lang][name[0]] == nil {					// 'param:temp' will populate the index parameter from the Param list
		return ""											// 'language:xx' will get the paramater for language xx
	}
	var item []string
	index := int64(-1)
	var err error
	if len(name) == 1 {
		return page.readLine(lang, name[0])
	} 
	for _, asdf := range(name[1:]) {
		if strings.HasPrefix(asdf,"param:") {
			param := strings.Split(asdf,":")
			index, err = strconv.ParseInt(page.Param[param[1]], 10, 64)
		} else if strings.HasPrefix(asdf,"language:") {
			lang = asdf[9:]
		} else {
			index, err = strconv.ParseInt(name[1], 10, 64)
		}
	}
	item = page.Data[lang][name[0]]
	if err != nil {
		return item[0]
	}
	return item[index]
}
func (page *Page) readLine(lang, name string) string {		// retrieves the entire line of text elements identified by that name
	whole := ""
	for _, s := range page.Data[lang][name] { whole += " "+s }
	return whole[1:]
}
func (page *Page) getHTML(data ...string) template.HTML { return template.HTML(page.data(data...)) }
func (page *Page) getCSS(data ...string) template.CSS { return template.CSS(page.data(data...)) }
func (page *Page) getScript(data ...string) template.JS { return template.JS(page.data(data...)) }
func (page *Page) service(data ...string) template.HTML {	// calls the service by its registered name
	return template.HTML(page.Site.Service[data[0]].Execute(activeSession, data[1:]))
}
func (page *Page) get(data ...string) Item {				// retireves an Item(interface{}) Object
	return page.Site.Service[data[0]].Get(page, activeSession, data[1:])
}
func (page *Page) getParam(name string) string {			// returns a page's named paramater
	if page.Param==nil || page.Param[name]=="" {
		return name
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
func (page *Page) getListByParam(name string) []string {			// returns a pages Data list via a page's paramater name
	lang := activeSession.GetLang()
	return page.Data[lang][page.Param[name]]
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
	setup := ""
	post := ""
	for _, d := range(data) {
		if strings.HasPrefix(d, "url:") { url = page.getParam(d[4:]) }
		if strings.HasPrefix(d, "handler:") { handler = d[8:] }
		if strings.HasPrefix(d, "target:") { target = d[7:] }
		if strings.HasPrefix(d, "trigger:") { trigger = d[8:] }
		if strings.HasPrefix(d, "data:") { jsData = page.getParam(d[5:]) }
		if strings.HasPrefix(d, "item:") { item = d[5:] }
		if strings.HasPrefix(d, "onclick:") { onClick = page.getParam(d[8:]) }
		if strings.HasPrefix(d, "var:") { variables += "var " + d[4:] + "; " }
		if strings.HasPrefix(d, "success:") { success = page.getParam(d[8:]) }
		if strings.HasPrefix(d, "setup:") { setup = "\n"+page.getParam(d[6:]) }
		if strings.HasPrefix(d, "post:") { post = "\n"+page.getParam(d[5:]) }
	}
	if success == "" {
		success = `var ul = $( "<ul/>", {"class": "my-new-list"});
			var obj = JSON.parse(data);	$("#`+target+`").empty(); $("#`+target+`").append(ul);
			$.each(obj, function(i,val) { item =`+item+`; `+onClick+` ul.append( item ); });`
	}
	return template.HTML(`<script>`+variables+`
		$(function() {
			$('#`+trigger+`-trigger').on('click', function() {`+
				setup+`
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
				});`+
				post+`
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
