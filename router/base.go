package router

import (
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/shengzhch/common/log"
	"net/http"
	"time"
)

type RouteCenter interface {
	GetRoute() []Handlewithlog
}

type Handlewithlog struct {
	name    string
	path    string
	methods []string
	*kithttp.Server
	*log.Logger
}

func (s Handlewithlog) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Server.ServeHTTP(w, r)
}

type Router struct {
	*mux.Router
}

func (router *Router) AddHandlers(h RouteCenter) {
	for _, r := range h.GetRoute() {
		if len(r.methods) < 1 {
			router.Handle(r.path, r)
		} else {
			router.Methods(r.methods...).Path(r.path).Handler(CorsHeaderSet(LoggerSet(r)))
		}
	}
}

func CorsHeaderSet(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		inner.ServeHTTP(w, r)
	})
}

func LoggerSet(hl Handlewithlog) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		hl.ServeHTTP(w, r)
		hl.Infof(
			"[%s] \t '%s' \t match[%s] \t %s",
			r.Method,
			r.RequestURI,
			hl.name,
			time.Since(start),
		)
	})
}

type ISeverDef interface {
	GetEP() endpoint.Endpoint
	DecReqFunc() kithttp.DecodeRequestFunc
	EncResFunc() kithttp.EncodeResponseFunc
	EncErrFunc() kithttp.ErrorEncoder
	GetServerOption() []kithttp.ServerOption
}

func NewHandle(name string, path string, methods []string, ie ISeverDef) Handlewithlog {
	s := kithttp.NewServer(ie.GetEP(), ie.DecReqFunc(), ie.EncResFunc(), ie.GetServerOption()...)
	return Handlewithlog{
		name:    name,
		path:    path,
		methods: methods,
		Server:  s,
		Logger:  log.GetALogger(),
	}
}
