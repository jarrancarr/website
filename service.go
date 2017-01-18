package website

import(
	"net/http"
	"io"
	"log"
)

var (
	logger *Log
)

type postFunc func(w http.ResponseWriter, r *http.Request, s *Session, p *Page) (string, error)
type filterFunc func(w http.ResponseWriter, r *http.Request, s *Session) (string, error)

type Item interface {}

type Service interface {
	Status() string
	Execute([]string, *Page) string
	Get(*Page, *Session, []string) Item
}

type Log struct {
	Trace, Debug, Info, Warning, Error *log.Logger
}

func NewLog(traceHandle io.Writer, debugHandle io.Writer, infoHandle io.Writer, warningHandle io.Writer, errorHandle io.Writer) *Log {
	logger = &Log{}
	logger.Trace = log.New(traceHandle, "TRACE: ",log.Ldate|log.Ltime|log.Lshortfile)
	logger.Debug = log.New(debugHandle, "DEBUG: ",log.Ldate|log.Ltime|log.Lshortfile)
	logger.Info = log.New(infoHandle, "INFO: ",log.Ldate|log.Ltime|log.Lshortfile)
	logger.Warning = log.New(warningHandle, "WARN: ",log.Ldate|log.Ltime|log.Lshortfile)
	logger.Error = log.New(errorHandle, "ERROR: ",log.Ldate|log.Ltime|log.Lshortfile)
	return logger
}