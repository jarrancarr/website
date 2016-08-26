package ecommerse

import (
	"github.com/jarrancarr/website"
	"net/http"
	"fmt"
	"io/ioutil"
	"strings"
)

type ECommerseService struct {
	index   []Category
	catalog map[string]map[string]Product
	acs *website.AccountService
}
func CreateService(acs *website.AccountService) *ECommerseService {
	ecs := ECommerseService{nil, nil, acs}
	return &ecs
}
func (ecs *ECommerseService) Status() string {
	return "good"
}
func (ecs *ECommerseService) Execute(session *website.Session, data []string) string {
	switch data[0] {
	case "get":
		return ""
	}
	return ""
}
func (ecs *ECommerseService) AddCategories(name, desc, image string) {
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
func (ecs *ECommerseService) GetCategories(w http.ResponseWriter, r *http.Request, s *website.Session) (string, error) {
	cats := "{"
	for indx, cat := range(ecs.index) {
		if indx > 0 { cats += ","	}
		cats += ` "` + cat.Name + `" : "` + cat.Description + `"`
	}
	w.Write([]byte(cats + " }"))
	return "ok", nil
} 
func (ecs *ECommerseService) GetProducts(w http.ResponseWriter, r *http.Request, s *website.Session) (string, error) {
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