package html

type HTMLer interface {
	AddClass() *HTMLer
	AddId() *HTMLer
	AddStyle() *HTMLer
}

type HTMLElement struct {
	class string
	id    string
	style string
}

type HTMLLink struct {
	Data string
	Url  string
	Cis  HTMLElement
}

func Link(data, url string) HTMLLink {
	return HTMLLink{data, url, HTMLElement{"", "", ""}}
}

func (elem *HTMLElement) AddClass(class string) {
	elem.class += " " + class
}
func (elem *HTMLElement) AddId(id string) {
	elem.id += " " + id
}
func (elem *HTMLElement) AddStyle(style string) {
	elem.style += " " + style
}

func (elem HTMLLink) Render() string {
	return elem.Cis.Render("u") + "</u>"
}
func (elem HTMLElement) Render(tag string) string {
	element := "<" + tag
	if elem.id != "" {
		element += " id=\"" + elem.id + "\" "
	}
	if elem.class != "" {
		element += " class=\"" + elem.class + "\" "
	}
	if elem.style != "" {
		element += " style=\"" + elem.style + "\" "
	}
	return element + ">"
}
