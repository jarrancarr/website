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
	"html/template"
	"strconv"

	"github.com/jarrancarr/website/html"
)

var ResourceDir = "../../"
var DataDir = "."

type Site struct {
	Name, Url	          	string
	SiteSessionCookieName 	string
	Tables                	*html.TableStow
	Html					*html.HTMLStow
	Pages                 	*PageStow
	UserSession           	map[string]*Session
	Service               	map[string]Service
	SiteProcessor         	map[string]postFunc
	ParamTriggerHandle    	map[string]postFunc
	Text                  	map[string][]string
	Data                  	map[string]interface{}
	Script                	map[string][]template.JS
	Param			    	map[string]string
}

func CreateSite(name, url string) *Site {
	site := Site{Name:name, Url:url, 
				SiteSessionCookieName:name + "-cookie",
				Tables:&html.TableStow{nil},
				UserSession:make(map[string]*Session),
				Text:make(map[string][]string), 
				Data:make(map[string]interface{}),
				Script:make(map[string][]template.JS),
				Html:&html.HTMLStow{nil}}
	return &site
}
func (site *Site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//err := site.home.tmpl.Execute(w, site.home)
	err := site.Pages.Ps["home"].tmpl.Execute(w, site.Pages.Ps["home"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
func (site *Site) GetCurrentSession(w http.ResponseWriter, r *http.Request) *Session {
	logger.Trace.Println("GetCurrentSession(http.ResponseWriter, r *http.Request)")
	sessionCookie, _ := r.Cookie(site.SiteSessionCookieName)
	//	cookies := r.Cookies()
	//	fmt.Printf("found %v cookies\n", len(cookies))
	//	for _, c := range cookies {
	//		fmt.Println(c.Name + " = " + c.Value)
	//	}
	if sessionCookie != nil && site.UserSession[sessionCookie.Value] != nil {
		logger.Trace.Println("Valid sessionCookie and user session:"+sessionCookie.Value)
		return site.UserSession[sessionCookie.Value]
	} else {
		logger.Trace.Println("creating new session");
		sessionKey := make([]byte, 16)
		rand.Read(sessionKey)
		http.SetCookie(w, &http.Cookie{site.SiteSessionCookieName, base64.URLEncoding.EncodeToString(sessionKey), "/",
			"localhost", time.Now().Add(time.Hour * 24), "", 50000, false, true, "none=none", []string{"none=none"}})
		site.UserSession[base64.URLEncoding.EncodeToString(sessionKey)] = createSession()
		site.UserSession[base64.URLEncoding.EncodeToString(sessionKey)].Data["name"] = "Anonymous"
		site.UserSession[base64.URLEncoding.EncodeToString(sessionKey)].Data["id"] = base64.URLEncoding.EncodeToString(sessionKey)

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
func (site *Site) AddPage(title, template, url string) *Page {		logger.Trace.Println();
	page, err := LoadPage(site, title, template, url)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if site.Pages == nil {
		site.Pages = &PageStow{nil}
	}
	if title == "" {		logger.Trace.Println(template);
		site.Pages.AddPage(template, page)
	} else {		logger.Trace.Println(title);
		site.Pages.AddPage(title, page)
	}
	return page
}
func (site *Site) AddService(name string, serve Service) Service {
	if site.Service == nil {
		site.Service = make(map[string]Service)
	}
	site.Service[name] = serve
	return serve
}
func (site *Site) AddScript(name, script string) *Site {
	site.Script[name] = append(site.Script[name], template.JS(script))
	return site
}
func (site *Site) AddBody(name, line string) *Site {
	if site.Text == nil {
		site.Text = make(map[string][]string)
	}
	site.Text[name] = []string{}
	quotes := false
	stringbuild := ""
	items := strings.Split(line, " ")
	for _, item := range items {
		if quotes {
			stringbuild += " " + item
			if strings.HasSuffix(item, "\"") {
				site.Text[name] = append(site.Text[name],stringbuild[:len(stringbuild)-1])
				quotes = false
			}
		} else if strings.HasPrefix(item, "\"") {
			quotes = true
			stringbuild = item[1:]
		} else {
			site.Text[name] = append(site.Text[name],item)
		}
	}
	return site
}
func (site *Site) AddParam(name, data string) *Site {
	if (site.Param==nil) {
		site.Param = make(map[string]string)
	}
	site.Param[name] = data
	return site
}
func (site *Site) AddParamTriggerHandler(name string, handle postFunc) *Site {
	if site.ParamTriggerHandle == nil {
		site.ParamTriggerHandle = make(map[string]postFunc)
	}
	site.ParamTriggerHandle[name] = handle
	return site
}
func (site *Site) Item(lang string, name ...string) template.CSS {
	var item []string
	var index int64
	var err error
	if len(name) == 1 {
		return template.CSS(site.fullBody(lang, name[0]))
	} 
	item = site.Text[name[0]]
	if strings.HasPrefix(name[1],"Body:") {
		index, err = strconv.ParseInt(site.Text[strings.Split(name[1],":")[1]][0], 10, 64)
	} else {
		index, err = strconv.ParseInt(name[1], 10, 64)
	}
	if err != nil {
		return template.CSS(item[0])
	}
	return template.CSS(item[index])
}
func (site *Site) GetHtml(name string) template.HTML {
	if site.Html == nil || site.Html.Hs[name] == nil {
		return ""
	}
	return template.HTML(site.Html.Hs[name][0].Render())
}
func (site *Site) GetScript(param ...string) []template.JS {
	if site.Script == nil {
		return nil
	}
	return site.Script[param[0]]
}
func (site *Site) fullBody(lang, name string) string {
	whole := ""
	for _, s := range site.Text[name] { whole += " "+s }
	return whole[1:]
}
func (site *Site) upload(w http.ResponseWriter, r *http.Request) {
	logger.Trace.Println("upload()")
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
	logger.Trace.Println("ServeResource()")
	path := ResourceDir + "/public" + r.URL.Path
	if strings.HasSuffix(r.URL.Path, "js") {
		w.Header().Add("Content-Type", "application/javascript")
	} else if strings.HasSuffix(r.URL.Path, "css") {
		w.Header().Add("Content-Type", "text/css")
	} else if strings.HasSuffix(r.URL.Path, "png") {
		w.Header().Add("Content-Type", "image/png+xml")
	} else if strings.HasSuffix(r.URL.Path, "svg") {
		w.Header().Add("Content-Type", "image/svg+xml")
	} else if strings.HasSuffix(r.URL.Path, "mp3") {
		w.Header().Add("Content-Type", "audio/mpeg")
	}

	data, err := ioutil.ReadFile(path)

	if err == nil {
		w.Write(data)
	} else {
		w.WriteHeader(404)
		w.Write([]byte("404, My Friend - " + http.StatusText(404)))
	}
}
