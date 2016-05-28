package html

type HTMLMenuItem struct {
	Name, Url string
	Cis       HTMLElement
}

type HTMLMenu struct {
	items []HTMLMenuItem
	Cis   HTMLElement
}

type MenuIndex struct {
	Mi map[string]*HTMLMenu
}

func (mi *MenuIndex) AddMenu(name string) {
	if mi.Mi == nil {
		mi.Mi = make(map[string]*HTMLMenu)
	}
	item := HTMLMenu{nil, HTMLElement{"", "", ""}}
	mi.Mi[name] = &item
}

func (menu *HTMLMenu) Add(class, id, style string) *HTMLMenu {
	if class != "" {
		menu.Cis.AddClass(class)
	}
	if id != "" {
		menu.Cis.AddId(id)
	}
	if style != "" {
		menu.Cis.AddStyle(style)
	}
	return menu
}
func (menu *HTMLMenu) AddItem(item *HTMLMenuItem) *HTMLMenu {
	if menu.items == nil {
		menu.items = make([]HTMLMenuItem, 0)
	}
	menu.items = append(menu.items, *item)
	return menu
}

func (menu *HTMLMenu) Render() string {
	element := menu.Cis.Render("ul")

	for _, m := range menu.items {
		element += m.Render()
	}
	return element + "</ul>"
}

func (mi *HTMLMenuItem) Render() string {
	return "<li role='presentation' class='active'><a href='" + mi.Name + "'>" + mi.Url + "</a></li>"
}
