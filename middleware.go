package grpcmux

import (
	"net/http"
	"slices"
)

type Middleware func(w http.ResponseWriter, r *http.Request, next http.Handler)

type mwSeq []Middleware

func (mws mwSeq) Build(term http.Handler) http.Handler {
	if len(mws) == 0 {
		return term
	}

	h := term
	for _, mw := range slices.Backward(mws) {
		next := h
		h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mw(w, r, next)
		})
	}

	return h
}
