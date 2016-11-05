package html

import (
	"strings"
)

type HTMLTag struct {
	tag, text string
	attributes map[string]string
	attributeList map[string][]string
	child []*HTMLTag
	//class, id, style string
}

type HTMLStow struct {
	Hs map[string]*HTMLTag
}

func (hs *HTMLStow) Add(name, tag string, data []string) *HTMLStow {
	hs.prep()
	hs.Hs[name] = NewTag(tag, "", "", "", "")
	for _, d := range(data) {
		if strings.Contains(d,":::") { 
			hs.Hs[name].AddAttribute(d[:strings.Index(d,":::")], d[strings.Index(d,":::")+3:])
		} else {
			hs.Hs[name].AppendText(d)
		}
	}
	return hs
}

func (hs *HTMLStow) AddTo(name, tag string, data []string) *HTMLStow {
	if hs.Hs == nil || hs.Hs[name] == nil {
		return hs
	}
	child := NewTag(tag, "", "", "", "")
	for _, d := range(data) {
		if strings.Contains(d,":::") { 
			child.AddAttribute(d[:strings.Index(d,":::")], d[strings.Index(d,":::")+3:])
		} else {
			child.AppendText(d)
		}
	}
	hs.Hs[name].AppendChild(child)
	return hs
}

func (hs *HTMLStow) Tag(name string, tag *HTMLTag) *HTMLStow {
	hs.prep()
	hs.Hs[name] = tag
	return hs
}

func (hs *HTMLStow) prep() {
	if hs.Hs == nil {
		hs.Hs = make(map[string]*HTMLTag)
	}
}


func (elem *HTMLTag) AddAttribute(name, value string) *HTMLTag {
	if elem.attributes == nil {
		elem.attributes = make(map[string]string)
	}
	elem.attributes[name] = value
	return elem
}
func (elem *HTMLTag) GetAttribute(name string) string {
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
func (elem *HTMLTag) RemoveAttribute(name string) *HTMLTag {
	elem.attributes[name] = ""
	return elem
}
func (elem HTMLTag) AddClass(value string) *HTMLTag {
	if elem.attributeList == nil {
		elem.attributeList = make(map[string][]string)
	}
	if elem.attributeList["class"] == nil {
		elem.attributeList["class"] = make([]string,2)
	} 
	elem.attributeList["class"] = append(elem.attributeList["class"], value)
	return &elem
}
func (elem *HTMLTag) RemoveClass(value string) *HTMLTag {
	for i,v := range(elem.attributeList["class"]) {
		if v == value {
			elem.attributeList["class"] = append(elem.attributeList["class"][:i],elem.attributeList["class"][i+1:]...)
			break;
		}
	}
	return elem
}
func (elem *HTMLTag) ReplaceText(text string) *HTMLTag {
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
func (elem HTMLTag) Render() string {
	element := "<" + elem.tag	
	for k,v := range(elem.attributes) {
		element += " "+k+"=\"" + v + "\" "
	}
	if elem.child == nil {
		return element + "/>"
	}
	element += ">"
	for _,c := range(elem.child) {
		element += c.Render()
	}
	return element + "</" + elem.tag + ">"
}

func NewTag(tag, id, class, style, text string) *HTMLTag {
	return &HTMLTag{tag, text, nil, nil, nil}
}
func NewTag2(tag, text string, attr []string) *HTMLTag {
	htmlTag := NewTag(tag, "", "", "", text)
	for _, d := range(attr) {
		if strings.Contains(d,":::") { 
			htmlTag.AddAttribute(d[:strings.Index(d,":::")], d[strings.Index(d,":::")+3:])
		} else {
			htmlTag.AppendText(d)
		}
	}
	return htmlTag
}
func MakeStow() *HTMLStow {
	return &HTMLStow{make(map[string]*HTMLTag)}
}