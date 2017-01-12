package html

import (
	//"fmt"
	"strings"
)

type HTMLTag struct {
	tag, text string
	attributes map[string]string
	attributeList map[string][]string
	child []*HTMLTag
}

type HTMLStow struct {
	Hs map[string][]*HTMLTag
}

func (hs *HTMLStow) Add(name string, tag *HTMLTag) *HTMLStow {
	hs.prep()
	hs.Hs[name] = append(hs.Hs[name],tag)
	return hs
}

func (hs *HTMLStow) Get(name string) []*HTMLTag {
	return hs.Hs[name]
}

func (hs *HTMLStow) prep() {
	if hs.Hs == nil {
		hs.Hs = make(map[string][]*HTMLTag)
	}
}

func (elem *HTMLTag) Attr(name, value string) *HTMLTag {
	if elem.attributeList[name] == nil {
		if elem.attributes == nil {
			elem.attributes = make(map[string]string)
		}
		if elem.attributes[name] == "" {
			elem.attributes[name] = value
		} else {
			if elem.attributeList == nil {
				elem.attributeList = make(map[string][]string)
			}
			elem.attributeList[name] = append(elem.attributeList[name],elem.attributes[name])
			elem.attributeList[name] = append(elem.attributeList[name],value)
			elem.attributes[name] = ""
		}
	} else {
		elem.attributeList[name] = append(elem.attributeList[name],value)
	}
	return elem
}
func (elem *HTMLTag) RemoveListAttribute(name, value string) *HTMLTag {
	// need to implement
	return elem
}
func (elem *HTMLTag) Get(name string) string {
	return elem.attributes[name]
}
func (elem *HTMLTag) GetId() string {
	return elem.attributes["id"]
}
func (elem *HTMLTag) GetClass() []string {
	return elem.attributeList["class"]
}
func (elem *HTMLTag) GetStyle() []string {
	return elem.attributeList["style"]
}
func (elem *HTMLTag) Class(value string) *HTMLTag {
	if elem.attributeList == nil {
		elem.attributeList = make(map[string][]string)
	}
	elem.attributeList["class"] = append(elem.attributeList["class"], value)
	return elem
}
func (elem *HTMLTag) Text(text string) *HTMLTag {
	elem.text = text
	return elem
}
func (elem *HTMLTag) AppendText(text string) *HTMLTag {
	elem.text += text
	return elem
}
func (elem *HTMLTag) AppendChild(tag *HTMLTag) *HTMLTag {
	if (elem.child == nil) {
		elem.child = make([]*HTMLTag,10)
	}
	elem.child = append(elem.child, tag)
	return elem
}
func (elem HTMLTag) String() string {
	return elem.Render();
}
func (elem HTMLTag) Render() string {
	element := "<" + elem.tag	
	for k,v := range(elem.attributes) {
		element += " "+k+"=\"" + v + "\""
	}	
	for k,v := range(elem.attributeList) {
		element += " "+k+"=\""
		for _,l := range(v) {
			element += l + " "
		}
		element += "\""
	}
	if elem.child == nil && elem.text == "" {
		return element + "/>"
	}
	element += ">"
	for _,c := range(elem.child) {
		if c != nil {
			element += c.Render()
		}
	}
	return element + elem.text + "</" + elem.tag + ">"
}

func NewTag3(tag, id, class, style, text string) *HTMLTag {
	t := &HTMLTag{tag, text, nil, nil, nil}
	t.Attr("id", id).Class(class).Attr("style",style)
	return t
}
func NewTag2(tag, text string, attr []string) *HTMLTag {
	htmlTag := NewTag(tag).Text(text)
	for _, d := range(attr) {
		if strings.Contains(d,":::") { 
			htmlTag.Attr(d[:strings.Index(d,":::")], d[strings.Index(d,":::")+3:])
		} else {
			htmlTag.AppendText(d)
		}
	}
	return htmlTag
}
func NewTag(html string) *HTMLTag {
	var id, class, style, text string
	var key, value []string
	token := strings.Split(html, " ")
	for _,t := range token[1:] {
		attr := strings.Split(t,"==")
		if len(attr) == 1 {
			text = attr[0]
		} else {
			switch(attr[0]) {
				case "id": id = attr[1] 
					break
				case "class": class = attr[1] 
					break
				case "style": style = attr[1] 
					break
				default: 
					key = append(key, attr[0])
					value = append(value, attr[1])
					break
			}
		}
	}
	
	htmlTag := NewTag3(token[0], id, class, style, text);
	for index, k := range(key) {
		htmlTag.Attr(k, value[index])
	}
	return htmlTag
}
func MakeStow() *HTMLStow {
	return &HTMLStow{make(map[string][]*HTMLTag)}
}