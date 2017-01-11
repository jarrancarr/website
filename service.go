package website

import(
	"net/http"
)

type postFunc func(w http.ResponseWriter, r *http.Request, s *Session, p *Page) (string, error)
type filterFunc func(w http.ResponseWriter, r *http.Request, s *Session) (string, error)

type Item interface {}

type Service interface {
	Status() string
	Execute([]string, *Page) string
	Get(*Page, *Session, []string) Item
}
