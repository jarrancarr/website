package service

type Service interface {
	Status() string
	Execute(string, string, string) string
}
