package middlewares

import (
	"log"
	"net/http"

	"github.com/go-park-mail-ru/2018_2_LSP_CHAT/webserver/handlers"
)

// Tracing Middleware for requests tracing
func Tracing(next handlers.HandlerFunc) handlers.HandlerFunc {
	return func(env *handlers.Env, w http.ResponseWriter, r *http.Request) error {
		log.Printf("Tracing request for %s", r.RequestURI)
		return next(env, w, r)
	}
}
