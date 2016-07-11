package website

type Service interface {
	Status() string
	Execute(*Session, []string) string
}
