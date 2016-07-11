package ecommerse

import (
	"github.com/jarrancarr/website"
)

type ECommerseService struct {
	index   []Category
	catalog map[string]*Product
}

func CreateService() website.Service {
	ecs := ECommerseService{nil, nil}
	return &ecs
}

func (ecs *ECommerseService) Status() string {
	return "good"
}

func (ecs *ECommerseService) Execute(session *website.Session, data []string) string {
	switch data[0] {
	case "get":
		return ecs.catalog[data[1]].Name
	}
	return ""
}
