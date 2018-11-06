package middlewares

import (
	"log"
	"net/http"

	"github.com/go-park-mail-ru/2018_2_LSP_CHAT/webserver/handlers"
)

// Logging Middleware for connection logging
func Logging(next handlers.HandlerFunc) handlers.HandlerFunc {
	return func(env *handlers.Env, w http.ResponseWriter, r *http.Request) error {
		log.Printf("Logged connection from %s", r.RemoteAddr)
		return next(env, w, r)
	}
}
