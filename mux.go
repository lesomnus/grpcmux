package grpcmux

import (
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/grpc"
)

type mux struct {
	gn http.Handler
	gw http.Handler
	h  http.Handler

	rsc map[string]bool
}

// Note that additional register will be ended with 404 for web.
func New(s *grpc.Server, opts ...Option) http.Handler {
	c := config{}
	for _, opt := range opts {
		opt(&c)
	}
	if c.http_handler == nil {
		c.http_handler = http.NotFoundHandler()
	}

	rsc := map[string]bool{}
	for name, info := range s.GetServiceInfo() {
		for _, method := range info.Methods {
			fullname := fmt.Sprintf("/%s/%s", name, method.Name)
			rsc[fullname] = true
		}
	}

	return mux{
		gn: s,
		gw: c.web_mws.Build(grpcWebHandler{s}),
		h:  c.web_mws.Build(c.http_handler),

		rsc: rsc,
	}
}

func (m mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var h http.Handler

	switch {
	case isGrpc(r):
		h = m.gn
	case isGrpcWebPreflight(r):
		fallthrough
	case isGrpcWeb(r):
		if m.rsc[r.URL.Path] {
			h = m.gw
		} else {
			h = http.NotFoundHandler()
		}
	default:
		h = m.h
	}

	h.ServeHTTP(w, r)
}

func isGrpc(r *http.Request) bool {
	return r.ProtoMajor >= 2 &&
		r.Method == http.MethodPost &&
		r.Header.Get("Content-Type") == "application/grpc"
}

func isGrpcWebPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions &&
		strings.Contains(strings.ToLower(r.Header.Get("Access-Control-Request-Headers")), "x-grpc-web")
}

func isGrpcWeb(r *http.Request) bool {
	return r.Method == http.MethodPost &&
		strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc-web")
}
