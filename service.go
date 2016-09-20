package website

type Item interface {}

type Service interface {
	Status() string
	Execute(*Session, []string) string
	Get(*Page, *Session, []string) Item
}
