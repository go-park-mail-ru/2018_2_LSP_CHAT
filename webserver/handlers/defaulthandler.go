package handlers

import (
	"net/http"
)

func DefaultHandler(env *Env, w http.ResponseWriter, r *http.Request) error {
	return StatusData{http.StatusMethodNotAllowed, map[string]string{"error": "Method is not allowed"}}
}
