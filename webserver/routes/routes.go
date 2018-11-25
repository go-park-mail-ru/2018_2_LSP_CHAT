package routes

import (
	"net/http"

	"github.com/go-park-mail-ru/2018_2_LSP_CHAT/webserver/handlers"
	"github.com/go-park-mail-ru/2018_2_LSP_CHAT/webserver/middlewares"
)

func Get() handlers.HandlersMap {
	handlersMap := handlers.HandlersMap{}
	handlersMap["/create"] = makeRequest(handlers.HandlersMap{
		"get": handlers.CreateNewChat,
	})
	handlersMap["/connect"] = makeRequest(handlers.HandlersMap{
		"get": handlers.ConnectToChat,
	})
	handlersMap["/chats"] = makeRequest(handlers.HandlersMap{
		"get": handlers.GetAllCHats,
	})
	handlersMap["/private"] = makeRequest(handlers.HandlersMap{
		"get": handlers.ConnectToPrivateChat,
	})
	handlersMap["/getprivatechat"] = makeRequest(handlers.HandlersMap{
		"get": handlers.GetPrivateChatMessages,
	})
	return handlersMap
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
