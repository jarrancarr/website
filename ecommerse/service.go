package ecommerse

import (
	"github.com/jarrancarr/website/service"
)

type ECommerseService struct {
	index   []Category
	catalog map[string]*Product
}

func CreateService() service.Service {
	ecs := ECommerseService{nil, nil}
	return &ecs
}

func (ecs *ECommerseService) Status() string {
	return "good"
}

func (ecs *ECommerseService) Execute(command, data string) string {
	switch command {
	case "get":
		return ecs.catalog[data].Name
	}
	return ""
}
