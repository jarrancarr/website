package website

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/jarrancarr/website/html"
)

type Page struct {
	Title, Url          string
	Text                map[string][]string
	Data                map[string]interface{}
	Param               map[string]string
	Html                *html.HTMLStow      // generic html tag snippets
	Site                *Site               // reference to site
	Parent              *Page               // parent page
	pages               *PageStow           // sub pages
	tables              *html.TableStow     // tables
	paramTriggerHandle  map[string]postFunc // functions executed with URL parameters
	postHandle          map[string]postFunc // functions executed from a post request
	ajaxHandle          map[string]postFunc // functions that respond to AJAX requests
	tmpl                *template.Template  // this pages HTML template
	initProcessor       []postFunc          // initial processors before site processors
	preProcessor        []postFunc          // processors after site processors
	postProcessor       []postFunc          // processors after page
	bypassSiteProcessor map[string]bool     // any site processor to not precess for this page
	ActiveSession       *Session
}

type PageStow struct {
	Ps map[string]*Page
}

var (
	pageLock = &sync.Mutex{}
)

func LoadPage(site *Site, title, tmplName, url string) (*Page, error) {
	var text map[string][]string
	if title != "" {
		filename := DataDir + "/" + title + ".txt"
		data, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		text = make(map[string][]string)
		r := bufio.NewReader(data)
		s, _, e := r.ReadLine()
		for e == nil {
			if string(s) != "" {
				if strings.HasPrefix(string(s), "<<lang>>") {
					//lang = string(s[8:])
					//body[lang] = make(map[string][]string)
					break
				} else {
					field := strings.Split(string(s), ">>")
					items := strings.Split(field[1], " ")
					quotes := false
					stringbuild := ""
					for _, item := range items {
						if quotes {
							stringbuild += " " + item
							if strings.HasSuffix(item, "\"") {
								text[field[0]] = append(text[field[0]], stringbuild[:len(stringbuild)-1])
								quotes = false
							}
						} else if strings.HasPrefix(item, "\"") {
							quotes = true
							stringbuild = item[1:]
						} else {
							text[field[0]] = append(text[field[0]], item)
						}
					}
				}
			}
			s, _, e = r.ReadLine()
		}
	}

	page := &Page{Title: title, Text: text, Site: site, Param: make(map[string]string), Html: &html.HTMLStow{nil}}
	page.tmpl = template.Must(template.New(tmplName + ".html").Funcs(
		template.FuncMap{
			"table":   page.table,
			"css":     page.css,
			"script":  page.Site.GetScript,
			"service": page.service,
			"get":     page.get,
			"page":    page.page,
			"debug":   page.debug,
			"html":    page.getHtml,
			"htmls":   page.getHtmls,
			"text":    page.text,
			"list":    page.list,
			"line":    page.line,
			"data":    page.data,
			"param":   page.getParam,
			"session": page.getSessionParam,
			"ajax":    page.ajax,
			"target":  page.target}).
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
	logger.Debug.Println("ServeHTTP(ttp.ResponseWriter, r *http.Request)")
	pageLock.Lock()
	page.ActiveSession = page.Site.GetCurrentSession(w, r)
	logger.Trace.Println("  running initProcessors")
	for _, pFunc := range page.initProcessor {
		status, err := pFunc(w, r, page.ActiveSession, page)
		if status != "ok" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			pageLock.Unlock()
			return
		}
	}
	logger.Trace.Println("  running SiteProcessors")
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
	logger.Trace.Println("  running preProcessors")
	for _, pFunc := range page.preProcessor {
		status, err := pFunc(w, r, page.Site.GetCurrentSession(w, r), page)
		if status != "ok" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			pageLock.Unlock()
			return
		}
	}
	paramMap := r.URL.Query()
	logger.Trace.Println("  processing param map")
	for key, _ := range paramMap {
		page.Param[key] = paramMap.Get(key)
	}
	for key, _ := range paramMap {
		if page.paramTriggerHandle[key] != nil {
			page.paramTriggerHandle[key](w, r, page.ActiveSession, page)
		}
		if page.Site.ParamTriggerHandle[key] != nil {
			page.Site.ParamTriggerHandle[key](w, r, page.ActiveSession, page)
		}
	}
	if r.Method == "POST" {
		logger.Debug.Println("  Method = POST")
		if page.postHandle[r.FormValue("postProcessingHandler")] == nil {
		} else {
			redirect, _ := page.postHandle[r.FormValue("postProcessingHandler")](w, r, page.Site.GetCurrentSession(w, r), page)
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
		logger.Debug.Println("  Method = AJAX")
		status, err := page.ajaxHandle[r.Header.Get("ajaxProcessingHandler")](w, r, page.Site.GetCurrentSession(w, r), page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		if status != "ok" {
			http.Redirect(w, r, status, 307)
		}
		pageLock.Unlock()
		return
	} else {
		logger.Debug.Println("  Method = GET")
		// A normal GET request
		err := page.tmpl.Execute(w, page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
	logger.Trace.Println("  running postProcessor")
	for _, pFunc := range page.postProcessor {
		status, err := pFunc(w, r, page.ActiveSession, page)
		if status != "ok" {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			pageLock.Unlock()
			return
		}
	}
	page.Site.GetCurrentSession(w, r).Data["navigation"] = r.RequestURI
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
	if page.Param == nil {
		page.Param = make(map[string]string)
	}
	page.Param[name] = data
	return page
}
func (page *Page) AddData(name string, data interface{}) *Page {
	if page.Data == nil {
		page.Data = make(map[string]interface{})
	}
	page.Data[name] = data
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
	if page.bypassSiteProcessor == nil {
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
			for i, d := range name { // add any context parameters
				if i < 1 {
					continue
				}
				pair := strings.Split(d, ">>")
				page.Site.Pages.Ps[name[0]].AddParam(pair[0], pair[1])
			}
			page.Site.Pages.Ps[name[0]].ActiveSession = page.ActiveSession
			return template.HTML(page.Site.Pages.Ps[name[0]].Render())
		}
	}
	for i, d := range name { // add any context parameters
		if i < 1 {
			continue
		}
		pair := strings.Split(d, ">>")
		page.pages.Ps[name[0]].AddParam(pair[0], pair[1])
	}
	page.pages.Ps[name[0]].ActiveSession = page.ActiveSession
	return template.HTML(page.pages.Ps[name[0]].Render())
}
func (page *Page) debug(name ...string) template.HTML {
	all := "<br/><div class='debug'><p><code>page: " + page.Title
	all += "<br/>&nbsp&nbspUrl: " + page.Url
	all += "<br/>&nbsp&nbspData: "
	all += "<br/>&nbsp&nbspparam: "
	for key, val := range page.Param {
		all += "<br/>&nbsp&nbsp&nbsp&nbsp" + key + ": " + val
	}
	all += "<br/>&nbsp&nbsppostHandle: "
	for key, _ := range page.postHandle {
		all += "<br/>&nbsp&nbsp&nbsp&nbsp" + key
	}
	all += "<br/>&nbsp&nbspajaxHandle: "
	for key, _ := range page.ajaxHandle {
		all += "<br/>&nbsp&nbsp&nbsp&nbsp" + key
	}
	all += "</code></p></div>"
	return template.HTML(all)
}

func (page *Page) getHtml(name ...string) []template.HTML {
	logger.Debug.Println("getHtml('"+strings.Join(name,"', '")+"')")
	length := page.params(name[1:]...)
	logger.Debug.Printf("length = %d",length)
	if page.Html == nil {
		if page.Parent == nil {
			return nil
		}
		return page.Parent.getHtml(name...)
	}
	var list []template.HTML
	if length > 0 {
		for i := 0 ; i<length ; i++ {
			page.Param["ITERATOR"] = fmt.Sprintf("%d", i)
			list = append(list, template.HTML(page.jsp(page.Html.Hs[name[0]][0].Render(), i)))
		}
		return list
	} else {
		return []template.HTML{template.HTML(page.jsp(page.Html.Hs[name[0]][-length].Render(), -1))}
	}
}
func (page *Page) getHtmls(name ...string) []template.HTML {
	logger.Trace.Println("getHtmls("+strings.Join(name,",")+")")
	length := page.params(name[1:]...)
	if page.Html == nil {
		logger.Debug.Println("page.Html is nil... checking parent")
		return page.Parent.getHtmls(name...)
	}
	var list []template.HTML
	for i := 0 ; i<length ; i++ {
		for _, ht := range page.Html.Hs[name[0]] {
			page.Param["ITERATOR"] = page.Param["ITERATOR"] + " " + fmt.Sprintf("%d", i)
			list = append(list, template.HTML(page.jsp(ht.Render(), i)))
		}
	}
	return list
}
func (page *Page) css(name ...string) template.CSS { // item pulls a string from the parameter text file by name and optionally a
	logger.Trace.Println("css("+strings.Join(name,",")+")")
	return template.CSS(page.text(name...)) // number indicating which index of that string to pull
}
func (page *Page) jsp(input string, index int) string {
	logger.Trace.Printf("jsp('%s',%d)",input,index)
	if start := strings.Index(input, "${"); start >= 0 {
		end := strings.Index(input[start:], "}")
		if index == -1 {
			return page.jsp(input[:start] + page.Param[input[start+2:start+end]] + input[start+end+1:], index)
		} else {
			logger.Trace.Println("jsp:mark-- "+input[start+2:start+end]+": "+page.Param[input[start+2:start+end]])
			return page.jsp(input[:start] + strings.Split(page.Param[input[start+2:start+end]]," ")[index] + input[start+end+1:], index)
		}
	}
	return input
}
func (page *Page) data(data string) interface{} {
	return page.Data[data]
}
func (page *Page) params(data ...string) int { // add any context parameters: returns the count of the minimum array size
	logger.Trace.Println("params("+strings.Join(data,",")+")")
	if len(data) == 0 { return 1 }
	minimum := 999;
	for _, d := range data { 
		pair := strings.SplitN(d, ">>",4)
		if len(pair) == 1 {
			num, err := strconv.ParseInt(data[1], 10, 64)
			if err == nil {
				return int(-num)
			}
		}
		if len(pair) == 2 {
			words := strings.SplitN(pair[1],",",-1)
			if len(words) < minimum { 
				minimum = len(words) 
			}
			page.AddParam(pair[0], strings.Join(words," "))
		}
		if len(pair) == 3 {
			words := strings.SplitN(pair[2],",",-1)
			if len(words) < minimum {
				minimum = len(words) 
			}
			page.pages.Ps[pair[0]].AddParam(pair[1], strings.Join(words," "))
		}
	}
	return minimum
}
func (page *Page) text(name ...string) string { // retrieves a data element as a string
	logger.Trace.Println("params("+strings.Join(name,",")+")")
	if page.Text[name[0]] == nil { // 'param:temp' will populate the index parameter from the Param list
		return "" // 'language:xx' will get the paramater for language xx
	}
	if len(name) == 1 {
		return page.line(name[0])
	}
	var item []string
	index := page.parse(name[1:]...)
	item = page.Text[name[0]]
	if len(index) == 0 {
		return item[0]
	}
	return item[index[0]]
}
func (page *Page) parse(data ...string) []int64 {
	logger.Trace.Println("params("+strings.Join(data,",")+")")
	var answ []int64	
	for _, asdf := range data {
		if strings.HasPrefix(asdf, "param:") {
			param := strings.Split(asdf, ":")
			index, err := strconv.ParseInt(page.Param[param[1]], 10, 64)
			if err == nil {
				answ = append(answ, index)
			}
		} else {
			index, err := strconv.ParseInt(data[1], 10, 64)
			if err == nil {
				answ = append(answ, index)
			}
		}
	}
	return answ
}
func (page *Page) list(name string) []string {
	return page.Text[name]
}
func (page *Page) line(name string) string { // retrieves the entire line of text elements identified by that name
	whole := ""
	for _, s := range page.Text[name] {
		whole += " " + s
	}
	return whole[1:]
}
func (page *Page) getCSS(data ...string) template.CSS     { return template.CSS(page.text(data...)) }
func (page *Page) getScript(data ...string) template.JS   { return template.JS(page.text(data...)) }
func (page *Page) service(data ...string) template.HTML { // calls the service by its registered name
	return template.HTML(page.Site.Service[data[0]].Execute(data[1:], page))
}
func (page *Page) get(data ...string) Item { // retireves an Item(interface{}) Object
	return page.Site.Service[data[0]].Get(page, page.ActiveSession, data[1:])
}
func (page *Page) getParam(name string) string { // returns a page's named paramater
	if page.Param == nil || page.Param[name] == "" {
		return name
	}
	return page.Param[name]
}
func (page *Page) getSessionParam(name string) string { // returns a session paramater
	if page.ActiveSession == nil {
		return "no session"
	}
	if name == "language" {
		return page.ActiveSession.GetLang()
	}
	return page.ActiveSession.Data[name]
}
func (page *Page) getTextByParam(name string) []string { // returns a pages Data list via a page's paramater name
	return page.Text[page.Param[name]]
}
func (page *Page) ajax(data ...string) template.HTML { // sets up an ajax call to retrieve data from the server.
	url := page.Url // this call should be accompanied by a target on the page
	handler := ""   // and the AJAX Handler function
	trigger := ""
	target := ""
	onClick := ""
	item := "$(document.createElement('li')).text( i + ' - ' + val )"
	jsData := "'greet':'hello there, partner!'"
	variables := ""
	success := ""
	setup := ""
	post := ""
	for _, d := range data {
		if strings.HasPrefix(d, "url:") {
			url = page.getParam(d[4:])
		}
		if strings.HasPrefix(d, "handler:") {
			handler = d[8:]
		}
		if strings.HasPrefix(d, "target:") {
			target = d[7:]
		}
		if strings.HasPrefix(d, "trigger:") {
			trigger = d[8:]
		}
		if strings.HasPrefix(d, "data:") {
			jsData = page.getParam(d[5:])
		}
		if strings.HasPrefix(d, "item:") {
			item = d[5:]
		}
		if strings.HasPrefix(d, "onclick:") {
			onClick = page.getParam(d[8:])
		}
		if strings.HasPrefix(d, "var:") {
			variables += "var " + d[4:] + "; "
		}
		if strings.HasPrefix(d, "success:") {
			success = page.getParam(d[8:])
		}
		if strings.HasPrefix(d, "setup:") {
			setup = "\n" + page.getParam(d[6:])
		}
		if strings.HasPrefix(d, "post:") {
			post = "\n" + page.getParam(d[5:])
		}
	}
	if success == "" {
		success = `var ul = $( "<ul/>", {"class": "my-new-list"});
			var obj = JSON.parse(data);	$("#` + target + `").empty(); $("#` + target + `").append(ul);
			$.each(obj, function(i,val) { item =` + item + `; ` + onClick + ` ul.append( item ); });`
	}
	return template.HTML(`<script>` + variables + `
		$(function() {
			$('#` + trigger + `-trigger').on('click', function() {` +
		setup + `
				$.ajax({
					url: '/` + url + `',
					type: 'AJAX',
					headers: { 'ajaxProcessingHandler':'` + handler + `' },
					dataType: 'html',
					data: { ` + jsData + ` },
					success: function(data, textStatus, jqXHR) {
						` + success + `	
					},
					error: function(data, textStatus, jqXHR) {
						console.log("button fail!");
					}
				});` +
		post + `
			});
		});
	</script>`)
}

//sets up a div target for the ajax call
func (page *Page) target(name string) template.HTML {
	return template.HTML("<div id='" + name + "'></div>")
}
func (page *Page) Render() template.HTML {
	buf := new(bytes.Buffer)
	page.tmpl.Execute(buf, page)
	return template.HTML(buf.String())
}
