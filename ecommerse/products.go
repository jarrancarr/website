package ecommerse

type Category struct {
	Name        string
	Description string
	ImageName   string
}

type Product struct {
	Name        string
	Description string
	ImageName   string
	Price       int
	InStock     int
}

type Order struct {
	Item     *Product
	Quantity int
}

type Cart struct {
	line []Order
}

func ProductNameList(list []Category) []string {
	names := make([]string, len(list))
	for idx, cate := range list {
		names[idx] = cate.Name
	}
	return names
}

func loadCategories(name string) ([]Category, error) {
	return nil, nil
}

func loadProducts(name string) ([]Product, error) {
	return nil, nil
}
