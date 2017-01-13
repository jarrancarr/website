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
	Line []*Order
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

func (cart *Cart) addOrder(ord *Order) {
	if cart.Line == nil {
		cart.Line = make([]*Order, 0)
	}
	newItem := true
	for _, i := range(cart.Line) {
		if i.Item.Name == ord.Item.Name {
			i.Quantity = i.Quantity + 1
			newItem = false
			break
		}
	}
	if newItem {
		cart.Line = append(cart.Line, ord)
	}
}

func (cart *Cart) Empty() {
	cart.Line = make([]*Order, 0)
}

func (p Product) Cost() float64 {
	return float64(p.Price)/100.0
}