package middlewares

import (
	"net/http"

	"github.com/go-park-mail-ru/2018_2_LSP_CHAT/webserver/handlers"
)

// Middleware Http middleware
type Middleware func(next handlers.HandlerFunc) handlers.HandlerFunc

// Chain Util for chaining different middlewares into new one
func Chain(mw ...Middleware) Middleware {
	return func(final handlers.HandlerFunc) handlers.HandlerFunc {
		return func(env *handlers.Env, w http.ResponseWriter, r *http.Request) error {
			last := final
			for i := len(mw) - 1; i >= 0; i-- {
				last = mw[i](last)
			}
			return last(env, w, r)
		}
	}
}
