package middlewares

import (
	"fmt"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-park-mail-ru/2018_2_LSP_CHAT/webserver/handlers"
	"github.com/gorilla/context"
)

// Auth Middleware for protecting urls from unauthorized users
func Auth(next handlers.HandlerFunc) handlers.HandlerFunc {
	return func(env *handlers.Env, w http.ResponseWriter, r *http.Request) error {
		signature, err := r.Cookie("signature")
		if err != nil {
			fmt.Println("Not found")
			return handlers.StatusData{http.StatusUnauthorized, map[string]string{"error": "No signature cookie found"}}
		}

		headerPayload, err := r.Cookie("header.payload")
		if err != nil {
			return handlers.StatusData{http.StatusUnauthorized, map[string]string{"error": "No headerPayload cookie found"}}
		}

		tokenString := headerPayload.Value + "." + signature.Value

		claims := jwt.MapClaims{}
		_, err = jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("abcderfs334f34r3we34"), nil
		})

		if err != nil {
			signatureCoookie := http.Cookie{
				Name:    "signature",
				Expires: time.Now().AddDate(0, 0, -1),
			}
			headerPayloadCookie := http.Cookie{
				Name:    "signature",
				Expires: time.Now().AddDate(0, 0, -1),
			}
			http.SetCookie(w, &signatureCoookie)
			http.SetCookie(w, &headerPayloadCookie)
			return handlers.StatusData{http.StatusUnauthorized, map[string]string{"error": err.Error()}}
		}

		context.Set(r, "claims", claims)

		return next(env, w, r)
	}
}
