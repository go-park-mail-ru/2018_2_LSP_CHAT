package handlers

import (
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/websocket"
	// "github.com/gorilla/websocket"
)

type WSMessage struct {
	command string
	data    string
}

var rooms = make(map[string]*ChatRoom)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleChatConnection(chatHash string, userID string, c *websocket.Conn) error {
	if _, ok := rooms[chatHash]; !ok {
		rooms[chatHash] = NewChatRoom()
		go rooms[chatHash].Run()
	}

	subscription := rooms[chatHash].Subscribe()
	defer rooms[chatHash].Unsubscribe(subscription)

	rooms[chatHash].Join(userID)
	defer rooms[chatHash].Leave(userID)

	// Send down the archive.
	for _, event := range subscription.Archive {
		if err := c.WriteJSON(event); err != nil {
			return nil
		}
	}

	newMessages := make(chan string)
	go func() {
		var msg string
		for {
			err := c.ReadJSON(&msg)
			if err != nil {
				close(newMessages)
				return
			}
			newMessages <- msg
		}
	}()

	for {
		select {
		case event := <-subscription.New:
			if c.WriteJSON(&event) != nil {
				// They disconnected.
				return nil
			}
		case msg, ok := <-newMessages:
			// If the channel is closed, they disconnected.
			if !ok {
				return nil
			}

			// Otherwise, say something.
			rooms[chatHash].Say(userID, msg)
		}
	}
}

func GetUpdgradeConnection(env *Env, w http.ResponseWriter, r *http.Request) error {
	chatHashURL, ok := r.URL.Query()["chat"]

	if !ok || len(chatHashURL[0]) < 1 {
		return StatusData{http.StatusBadRequest, map[string]string{"error": "You must specify chat ID"}}
	}
	chatHash := chatHashURL[0]

	claims := context.Get(r, "claims").(jwt.MapClaims)
	userID := claims["id"].(string)

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	defer c.Close()

	handleChatConnection(chatHash, userID, c)
	if rooms[chatHash].users == 0 {
		delete(rooms, chatHash)
	}

	return StatusData{http.StatusOK, map[string]string{"error": "err"}}
}
