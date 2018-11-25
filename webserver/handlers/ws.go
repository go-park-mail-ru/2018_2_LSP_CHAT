package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-park-mail-ru/2018_2_LSP_CHAT/user"
	"github.com/gorilla/context"
	"github.com/gorilla/websocket"
)

var currUserID = 0

type WSMessage struct {
	command string
	data    string
}

var rooms = make(map[int]*ChatRoom)

// var privateRooms = make(map[int]*ChatRoom)
var privateSockets = make(map[int]*websocket.Conn)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handlePrivateChatConnection(env *Env, u *user.User, c *websocket.Conn) error {
	newCommands := make(chan Command)
	go func() {
		var cmd Command
		for {
			err := c.ReadJSON(&cmd)
			if err != nil {
				close(newCommands)
				return
			}
			newCommands <- cmd
		}
	}()

	for {
		select {
		// case event := <-subscription.New:
		// 	if c.WriteJSON(&event) != nil {
		// 		return nil
		// 	}
		case cmd, ok := <-newCommands:
			if !ok {
				return nil
			}
			switch cmd.Action {
			case "message":
				_, ok := cmd.Params["message"]
				if !ok {
					return nil
				}
				_, ok = cmd.Params["user"]
				if !ok {
					return nil
				}
				userID, err := strconv.Atoi(cmd.Params["user"])
				if err != nil {
					return nil
				}
				_, err = env.DB.Query("INSERT INTO private_messages (author, to, text) VALUES ($1, $2, $3)", u.ID, userID, cmd.Params["message"])
				if err != nil {
					return nil
				}
				event := newEvent("message", strconv.Itoa(u.ID), cmd.Params["message"])
				if privateSockets[userID].WriteJSON(&event) != nil {
					return nil
				}
				if privateSockets[u.ID].WriteJSON(&event) != nil {
					return nil
				}
				// case "list":
				// 	_, ok = cmd.Params["user"]
				// 	if !ok {
				// 		return nil
				// 	}
				// 	userID, err := strconv.Atoi(cmd.Params["user"])
				// 	if err != nil {
				// 		return nil
				// 	}
				// 	_, err = env.DB.Query("SELECT author, text, date_created FROM private_messages WHERE author = $1 AND to = $2 OR author = $2 AND to = $1 ORDER BY date_created", u.ID, userID, cmd.Params["message"])
				// 	if err != nil {
				// 		return nil
				// 	}

				// 	event := newEvent("message", strconv.Itoa(u.ID), cmd.Params["message"])
				// 	if privateSockets[userID].WriteJSON(&event) != nil {
				// 		return nil
				// 	}
				// 	if privateSockets[u.ID].WriteJSON(&event) != nil {
				// 		return nil
				// 	}
			}
		}
	}
}

func handleChatConnection(env *Env, chat *ChatRoom, u *user.User, c *websocket.Conn) error {
	subscription := chat.Subscribe()
	defer chat.Unsubscribe(subscription)

	chat.Join(u)
	defer chat.Leave(u)

	archive, err := chat.GetArchive(env.DB)
	if err != nil {
		return err
	}

	for _, msg := range archive {
		if err := c.WriteJSON(msg); err != nil {
			return err
		}
	}

	newCommands := make(chan Command)
	go func() {
		var cmd Command
		for {
			err := c.ReadJSON(&cmd)
			if err != nil {
				close(newCommands)
				return
			}
			newCommands <- cmd
		}
	}()

	for {
		select {
		case event := <-subscription.New:
			if c.WriteJSON(&event) != nil {
				return nil
			}
		case cmd, ok := <-newCommands:
			if !ok {
				return nil
			}
			err := chat.Execute(env, u, cmd)
			if err != nil {
				return nil
			}
			// switch cmd.Action {
			// 	var err error
			// 	if chat.private {
			// 		_, err = env.DB.Query("INSERT INTO private_messages (room_id, author, text) VALUES ($1, $2, $3)", chat.id, u.ID, msg)
			// 	} else {
			// 		_, err = env.DB.Query("INSERT INTO messages (room_id, author, text) VALUES ($1, $2, $3)", chat.id, u.ID, msg)
			// 	}
			// 	if err != nil {
			// 		return err
			// 	}
			// 	chat.Say(u, msg)
			// }
		}
	}
}

func ConnectToChat(env *Env, w http.ResponseWriter, r *http.Request) error {
	var u user.User
	u.ID = currUserID
	currUserID++

	claims := context.Get(r, "claims")
	if claims != nil {
		u.ID = int(claims.(jwt.MapClaims)["id"].(float64))
	}

	chatIDURL, ok := r.URL.Query()["id"]

	if !ok || len(chatIDURL[0]) < 1 {
		return StatusData{http.StatusBadRequest, map[string]string{"error": "You must specify chat ID"}}
	}
	chatID, err := strconv.Atoi(chatIDURL[0])
	if err != nil {
		fmt.Println(err)
		return StatusData{http.StatusBadRequest, map[string]string{"error": "You must specify numeric chat ID"}}
	}

	_, exists := rooms[chatID]
	if !exists {
		rows, err := env.DB.Query("SELECT id, password FROM rooms WHERE id = $1", chatID)
		if err != nil {
			fmt.Println(err)
			return StatusData{http.StatusInternalServerError, map[string]string{"error": "Internal system error"}}
		}
		if !rows.Next() {
			return StatusData{http.StatusBadRequest, map[string]string{"error": "Chat doesn't exist"}}
		}
		room := NewChatRoom(nil, nil, nil)

		var password sql.NullString
		err = rows.Scan(&room.id, &password)
		if err != nil {
			fmt.Println(err)
			return StatusData{http.StatusInternalServerError, map[string]string{"error": "Internal system error"}}
		}

		if temp, err := password.Value(); err == nil && temp != nil {
			room.password = temp.(string)
			room.protected = true
		}
		rooms[chatID] = room
	}

	if rooms[chatID].protected {
		passwordURL, ok := r.URL.Query()["password"]
		if !ok || len(passwordURL[0]) < 1 {
			return StatusData{http.StatusBadRequest, map[string]string{"error": "You must specify password for connecting to this chat"}}
		}
		if passwordURL[0] != rooms[chatID].password {
			return StatusData{http.StatusBadRequest, map[string]string{"error": "Wrong password"}}
		}
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return StatusData{http.StatusOK, map[string]string{"error": err.Error()}}
	}
	defer c.Close()

	err = handleChatConnection(env, rooms[chatID], &u, c)
	fmt.Println(err)
	if rooms[chatID].users == 0 {
		delete(rooms, chatID)
	}

	if err != nil {
		return StatusData{http.StatusOK, map[string]string{"error": err.Error()}}
	} else {
		return StatusData{http.StatusOK, map[string]string{}}
	}
}

func CreateNewChat(env *Env, w http.ResponseWriter, r *http.Request) error {
	var u user.User
	u.ID = currUserID
	currUserID++

	claims := context.Get(r, "claims")
	if claims != nil {
		u.ID = int(claims.(jwt.MapClaims)["id"].(float64))
	}

	chatTitleURL, hasTitle := r.URL.Query()["title"]
	chatPasswordURL, hasPassword := r.URL.Query()["password"]

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer c.Close()

	var chatTitle *string
	var chatPassword *string

	if hasTitle {
		chatTitle = &chatTitleURL[0]
	}
	if hasPassword {
		chatPassword = &chatPasswordURL[0]
	}

	chat := NewChatRoom(env.DB, chatPassword, chatTitle)
	if chat == nil {
		return StatusData{http.StatusBadRequest, map[string]string{"error": "Can't create chat due to error"}}
	}

	rooms[chat.id] = chat
	go chat.Run()

	err = handleChatConnection(env, chat, &u, c)
	if chat.users == 0 {
		delete(rooms, chat.id)
	}
	if err != nil {
		return StatusData{http.StatusOK, map[string]string{"error": err.Error()}}
	} else {
		return StatusData{http.StatusOK, map[string]string{}}
	}
}

func GetAllCHats(env *Env, w http.ResponseWriter, r *http.Request) error {
	result := make([]chatEntry, 0)
	for room := range rooms {
		result = append(result, chatEntry{room, rooms[room].users, rooms[room].title, rooms[room].protected})
	}
	return StatusData{http.StatusOK, map[string][]chatEntry{"chats": result}}
}

func ConnectToPrivateChat(env *Env, w http.ResponseWriter, r *http.Request) error {
	var u user.User
	claims := context.Get(r, "claims").(jwt.MapClaims)
	u.ID = int(claims["id"].(float64))

	_, exists := privateSockets[u.ID]
	if exists {
		return StatusData{http.StatusBadRequest, map[string]string{"error": "Already connected"}}
	}

	// anotherUserURL, ok := r.URL.Query()["user"]
	// if !ok {
	// 	return StatusData{http.StatusBadRequest, map[string]string{"error": "You need to specify another user ID"}}
	// }

	// anotherUserID, err := strconv.Atoi(anotherUserURL[0])

	// if err != nil {
	// 	return StatusData{http.StatusBadRequest, map[string]string{"error": "You need to specify numeric user ID"}}
	// }

	// id1 := 0
	// id2 := 0
	// if u.ID > anotherUserID {
	// 	id1 = u.ID
	// 	id2 = anotherUserID
	// } else {
	// 	id1 = anotherUserID
	// 	id2 = u.ID
	// }

	// rows, err := env.DB.Query("SELECT id, user1, user2 FROM private_rooms WHERE user1 = $1 AND user2 = $2 LIMIT 1", id1, id2)
	// if err != nil {
	// 	return StatusData{http.StatusInternalServerError, map[string]string{"error": "Internal system error"}}
	// }
	// roomID := 0
	// if rows.Next() {
	// 	room := NewPrivateChatRoom(id1, id2, nil)
	// 	err = rows.Scan(&room.id)
	// 	if err != nil {
	// 		return StatusData{http.StatusInternalServerError, map[string]string{"error": "Internal system error"}}
	// 	}
	// 	roomID = room.id
	// 	privateRooms[roomID] = room
	// } else {
	// 	room := NewPrivateChatRoom(id1, id2, env.DB)
	// 	if room == nil {
	// 		return StatusData{http.StatusInternalServerError, map[string]string{"error": "Internal system error"}}
	// 	}
	// 	roomID = room.id
	// 	privateRooms[roomID] = room
	// }

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	defer c.Close()

	privateSockets[u.ID] = c

	err = handlePrivateChatConnection(env, &u, c)

	// err = handleChatConnection(env, privateRooms[roomID], &u, c)

	return StatusData{http.StatusOK, map[string]string{"error": err.Error()}}
}

func GetPrivateChatMessages(env *Env, w http.ResponseWriter, r *http.Request) error {
	var u user.User
	claims := context.Get(r, "claims").(jwt.MapClaims)
	u.ID = int(claims["id"].(float64))

	toURL, hasTo := r.URL.Query()["to"]
	if !hasTo {
		return StatusData{http.StatusBadRequest, map[string]string{"error": "You need to specify to"}}
	}

	to, err := strconv.Atoi(toURL[0])
	if err != nil {
		return StatusData{http.StatusBadRequest, map[string]string{"error": "You need to specify numeric to"}}
	}

	rows, err := env.DB.Query("SELECT id, author, text, date_created FROM private_messages WHERE author = $1 AND to = $2 OR author = $2 AND to = $1 ORDER BY date_created", u.ID, to)
	if err != nil {
		return StatusData{http.StatusInternalServerError, map[string]string{"error": err.Error()}}
	}
	res := make([]historyEntry, 0)
	for rows.Next() {
		entry := historyEntry{}
		err = rows.Scan(&entry.ID, &entry.Author, &entry.Text, &entry.DateCreated)
		if err != nil {
			return StatusData{http.StatusInternalServerError, map[string]string{"error": err.Error()}}
		}
		res = append(res, entry)
	}
	return StatusData{http.StatusOK, map[string][]historyEntry{"messages": res}}
}
