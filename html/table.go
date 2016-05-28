package html

import (
	"errors"
)

type HTMLTable struct {
	header []HTMLCell
	row    []HTMLRow
	cis    HTMLElement
}

type HTMLRow struct {
	cell []HTMLCell
	cis  HTMLElement
}

type HTMLCell struct {
	data string
	cis  HTMLElement
}

type TableIndex struct {
	Ti map[string]*HTMLTable
}

func MakeTable(headers []string, class, id, style string) *HTMLTable {
	table := HTMLTable{nil, nil, HTMLElement{class, id, style}}
	table.header = make([]HTMLCell, len(headers))
	for idx, head := range headers {
		table.header[idx] = HTMLCell{head, HTMLElement{"", "", ""}}
	}
	table.row = make([]HTMLRow, 0, 10)
	return &table
}

func (ti *TableIndex) AddTable(name string, headers, data []string) {
	if ti.Ti == nil {
		ti.Ti = make(map[string]*HTMLTable)
	}
	ti.Ti[name] = MakeTable(headers, "", "", "")
	for elem := 0; elem < len(data); elem += len(headers) {
		ti.Ti[name].AddRow(data[elem : elem+len(headers)])
	}
}

func (table *HTMLTable) AddRow(data []string) error {
	if len(data) != len(table.header) {
		return errors.New("incorrect number of data elements")
	}
	table.row = append(table.row, HTMLRow{nil, HTMLElement{"", "", ""}})
	table.row[len(table.row)-1].cell = make([]HTMLCell, 0)
	for _, td := range data {
		table.row[len(table.row)-1].cell = append(table.row[len(table.row)-1].cell, HTMLCell{td, HTMLElement{"", "", ""}})
	}
	return nil
}

func (table *HTMLTable) AddClass(class string) *HTMLTable {
	table.cis.AddClass(class)
	return table
}
func (table *HTMLTable) AddId(id string) *HTMLTable {
	table.cis.AddId(id)
	return table
}
func (table *HTMLTable) AddStyle(style string) *HTMLTable {
	table.cis.AddStyle(style)
	return table
}
func (table *HTMLTable) AddRowClass(idx int, class string) *HTMLTable {
	if idx < len(table.row) {
		table.row[idx].cis.AddClass(class)
	}
	return table
}
func (table *HTMLTable) AddRowId(idx int, id string) *HTMLTable {
	if idx < len(table.row) {
		table.row[idx].cis.AddId(id)
	}
	return table
}
func (table *HTMLTable) AddRowStyle(idx int, style string) *HTMLTable {
	if idx < len(table.row) {
		table.row[idx].cis.AddStyle(style)
	}
	return table
}

func (table *HTMLTable) AddCellClass(row, column int, class string) *HTMLTable {
	if row < len(table.row) && column < len(table.header) {
		table.row[row].cell[column].cis.AddClass(class)
	}
	return table
}
func (table *HTMLTable) AddCellId(row, column int, id string) *HTMLTable {
	if row < len(table.row) && column < len(table.header) {
		table.row[row].cell[column].cis.AddId(id)
	}
	return table
}
func (table *HTMLTable) AddCellStyle(row, column int, style string) *HTMLTable {
	if row < len(table.row) && column < len(table.header) {
		table.row[row].cell[column].cis.AddStyle(style)
	}
	return table
}

func (table *HTMLTable) Render() string {
	element := table.cis.Render("table")
	element += "<tr>"
	for _, d := range table.header {
		element += d.Render()
	}
	element += "</tr>"
	for _, d := range table.row {
		element += d.Render()
	}
	element += "</table>"
	return element
}
func (elem HTMLRow) Render() string {
	element := elem.cis.Render("tr")
	for _, d := range elem.cell {
		element += d.Render()
	}
	return element + "</tr>"
}
func (elem HTMLCell) Render() string {
	return elem.cis.Render("td") + elem.data + "</td>"
}
