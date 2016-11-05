package html

import (
	//"errors"
	//"fmt"
	"strconv"
)

type HTMLTable struct {
	table *HTMLTag
	header *HTMLTag 
	row []*HTMLTag
}

type TableStow struct {
	Ts map[string]*HTMLTable
}

func (ts *TableStow) AddTable(name string, headers, data []string) {
	if ts.Ts == nil {
		ts.Ts = make(map[string]*HTMLTable)
	}
	ts.Ts[name] = MakeTable(headers, "", "", "")
	for elem := 0; elem < len(data); elem += len(headers) {
		ts.Ts[name].AddRow(data[elem : elem+len(headers)])
	}
}
func (t *HTMLTable) AddRow(data []string) {
	newRow := NewTag("tr", t.table.GetId()+"-row-"+strconv.Itoa(len(t.row)), "", "", "")
	t.row = append(t.row, newRow)
	for _, td := range(data) {
		newRow.AppendChild(NewTag("td","","","",td))
	}
}
func (t *HTMLTable) Render() string { return t.table.Render() }

func MakeTable(headers []string, class, id, style string) *HTMLTable {
	table := HTMLTable{NewTag("table", id, class, style, ""), NewTag("head", id+"-header", "", "", ""), make([]*HTMLTag,len(headers))}
	for idx, head := range headers {
		table.header.AppendChild(NewTag("td", id+"-header-"+strconv.Itoa(idx), "", style, head))
	}	
	return &table
}