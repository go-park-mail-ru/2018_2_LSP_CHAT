package routes

import (
	"net/http"

	"github.com/go-park-mail-ru/2018_2_LSP_CHAT/webserver/handlers"
	"github.com/go-park-mail-ru/2018_2_LSP_CHAT/webserver/middlewares"
)

func Get() handlers.HandlersMap {
	handlersMap := handlers.HandlersMap{}
	handlersMap["/game"] = makeRequest(handlers.HandlersMap{
		"get": middlewares.Cors(middlewares.Auth(handlers.GetUpdgradeConnection)),
	})
	return handlersMap
}

type CRUDHandler struct {
	PostHandler   handlers.HandlerFunc
	GetHandler    handlers.HandlerFunc
	PutHandler    handlers.HandlerFunc
	DeleteHandler handlers.HandlerFunc
}

func makeRequest(handlersMap handlers.HandlersMap) handlers.HandlerFunc {
	return func(env *handlers.Env, w http.ResponseWriter, r *http.Request) error {
		switch r.Method {
		case http.MethodGet:
			if _, ok := handlersMap["get"]; ok {
				return handlersMap["get"](env, w, r)
			} else {
				return middlewares.Cors(handlers.DefaultHandler)(env, w, r)
			}
		case http.MethodPost:
			if _, ok := handlersMap["post"]; ok {
				return handlersMap["post"](env, w, r)
			} else {
				return middlewares.Cors(handlers.DefaultHandler)(env, w, r)
			}
		case http.MethodPut:
			if _, ok := handlersMap["put"]; ok {
				return handlersMap["put"](env, w, r)
			} else {
				return middlewares.Cors(handlers.DefaultHandler)(env, w, r)
			}
		case http.MethodDelete:
			if _, ok := handlersMap["delete"]; ok {
				return handlersMap["delete"](env, w, r)
			} else {
				return middlewares.Cors(handlers.DefaultHandler)(env, w, r)
			}
		default:
			return middlewares.Cors(handlers.DefaultHandler)(env, w, r)
		}
	}
}

func makeCRUDHandler(handlersMap handlers.HandlersMap) CRUDHandler {
	var handler CRUDHandler
	if _, ok := handlersMap["post"]; ok {
		handler.PostHandler = handlersMap["post"]
	} else {
		handler.PostHandler = handlers.DefaultHandler
	}
	if _, ok := handlersMap["get"]; ok {
		handler.GetHandler = handlersMap["get"]
	} else {
		handler.GetHandler = handlers.DefaultHandler
	}
	if _, ok := handlersMap["put"]; ok {
		handler.PutHandler = handlersMap["put"]
	} else {
		handler.PutHandler = handlers.DefaultHandler
	}
	if _, ok := handlersMap["delete"]; ok {
		handler.DeleteHandler = handlersMap["delete"]
	} else {
		handler.DeleteHandler = handlers.DefaultHandler
	}
	return handler
}
