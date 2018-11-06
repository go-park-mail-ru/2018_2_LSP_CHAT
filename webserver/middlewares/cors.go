package middlewares

import (
	"net/http"

	"github.com/go-park-mail-ru/2018_2_LSP_CHAT/webserver/handlers"
)

// Cors Middleware that enables CORS
func Cors(next handlers.HandlerFunc) handlers.HandlerFunc {
	return func(env *handlers.Env, w http.ResponseWriter, r *http.Request) error {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")

		if r.Method == http.MethodOptions {
			return nil
		}

		w.Header().Set("Content-Type", "application/json")

		return next(env, w, r)
	}
}
