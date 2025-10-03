package grpcmux

import (
	"net/http"
)

type config struct {
	http_handler http.Handler
	web_mws      mwSeq
}

type Option func(m *config)

func WithHttpHandler(h http.Handler) Option {
	return func(c *config) {
		c.http_handler = h
	}
}

func WithWebMiddleware(mws ...Middleware) Option {
	return func(c *config) {
		c.web_mws = append(c.web_mws, mws...)
	}
}
