package ecommerse

import (
	"github.com/jarrancarr/website"
	"net/http"
	"fmt"
	"io/ioutil"
	"strings"
	"strconv"
)

type ECommerseService struct {
	index   []Category
	catalog map[string]map[string]Product
	acs *website.AccountService
	pages *website.PageIndex
}
func CreateService(acs *website.AccountService) *ECommerseService {
	ecs := ECommerseService{nil, nil, acs, nil}
	return &ecs
}
func (ecs *ECommerseService) Status() string {
	return "good"
}
func (ecs *ECommerseService) Execute(session *website.Session, data []string) string {
	switch data[0] {
	case "product":
		page := ecs.pages.Pi[data[1]]
		product := ecs.catalog[data[2]][data[3]]
		page.ClearData(data[2])
		page.AddData(data[2],product.Description)
		page.AddData(data[2],product.ImageName)
		page.AddData(data[2],fmt.Sprintf("%.2f",float32(product.Price)/100.0))
		return ""
	case "cart":
		cart := getCart(session)
		if data[1] == "count" {
			return fmt.Sprintf("%d",len(cart.Line))
		} else if data[1] == "isEmpty" {
			if len(cart.Line) == 0 {
				return "true"
			} else {
				return "false"
			}
		}
	}
	return ""
}
func (ecs *ECommerseService) Get(page *website.Page, session *website.Session, data []string) website.Item {
	switch data[0] {
	case "cart":
		cart := getCart(session)
		return cart
	}
	t := "Duke"
	n := "Bingo"
	d := "The Man!"
	return struct {
			Title, Name, Desc string
		} {
			t, n, d,
		}
}
func (ecs *ECommerseService) AddCategory(name, desc, image string) {
	if ecs.index == nil {
		ecs.index = make([]Category,0)
	}
	ecs.index = append(ecs.index, Category{name, desc, image})
}
func (ecs *ECommerseService) AddProduct(category, name, desc, image string, price, stock int) {
	if ecs.catalog == nil {
		ecs.catalog = make(map[string]map[string]Product)
	}
	if ecs.catalog[category] == nil {
		ecs.catalog[category] = make(map[string]Product)
	}
	ecs.catalog[category][name] = Product{name, desc, image, price, stock}
}
func (ecs *ECommerseService) GetCategories(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	cats := "{"
	for indx, cat := range(ecs.index) {
		if indx > 0 { cats += ","	}
		cats += ` "` + cat.Name + `" : "` + cat.Description + `"`
	}
	w.Write([]byte(cats + " }"))
	return "ok", nil
} 
func (ecs *ECommerseService) GetProducts(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	httpData, _ :=ioutil.ReadAll(r.Body)
	requestedCategory := strings.Split(string(httpData),"=")[1]
	fmt.Println("requestedCategory= "+requestedCategory)
	prods := "{"
	productList := ecs.catalog[requestedCategory]
	for name, prod := range(productList) {
		prods += ` "` + name + `" : "` + prod.Description + `", `
	}
	if len(prods) > 2 {
		w.Write([]byte(prods[:len(prods)-2] + " }"))
	} else {
		w.Write([]byte(prods + " }"))
	}
	return "ok", nil
}
func (ecs *ECommerseService) AddPage(name string, page *website.Page) *ECommerseService {
	if ecs.pages == nil { ecs.pages = &website.PageIndex{nil}	}
	ecs.pages.AddPage(name, page)
	return ecs
}
func (ecs *ECommerseService) AddToCart(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	cart := getCart(s)
	lang := s.GetLang()
	category, _ := strconv.Atoi(p.Param["category"])
	item := ecs.catalog[p.Body[lang]["Category"][category]][r.FormValue("product")]
	cart.addOrder(&Order{&item, 1})
	return "", nil
}
func (ecs *ECommerseService) ClearCart(w http.ResponseWriter, r *http.Request, s *website.Session, p *website.Page) (string, error) {
	cart := getCart(s)
	cart.Empty()
	return "",nil
}
func getCart(s *website.Session) *Cart {
	var cart *Cart 
	obj := s.Item["cart"]
	if obj == nil {
		cart = &Cart{}
		s.Item["cart"] = cart
	} else {
		cart = obj.(*Cart)
	}
	return cart
}