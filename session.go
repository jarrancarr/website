package website

type Session struct {
	Item map[string]interface{}
	Data map[string]string
}

func (s *Session) GetLang() string {
	lang := s.Data["language"]
	if lang == "" {
		return "en"
	} 
	return lang
}
func createSession() *Session {
	return &Session{make(map[string]interface{}), make(map[string]string)}
}
func (s *Session) AddItem(name string, item interface{}) {
	if s.Item==nil {
		s.Item = make(map[string]interface{})
	}
	s.Item[name] = item
}
func (s *Session) GetItem(name string) interface{} {
	return s.Item[name]
}
func (s *Session) AddData(name, data string) {
	if s.Data==nil {
		s.Data = make(map[string]string)
	}
	s.Data[name] = data
}
func (s *Session) GetData(name string) string {
	return s.Data[name]
}
func (s *Session) GetId() string {
	return s.Data["id"]
}
func (s *Session) GetUserName() string {
	return s.Data["name"]
}
