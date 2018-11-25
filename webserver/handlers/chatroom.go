package handlers

import (
	"container/list"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2018_2_LSP_CHAT/user"
)

type ChatRoom struct {
	subscribe   chan (chan<- Subscription)
	unsubscribe chan (<-chan Event)
	publish     chan Event
	users       int
	title       string
	password    string
	protected   bool
	id          int
	private     bool
	privateData struct {
		user1 int
		user2 int
	}
}

func NewChatRoom(db *sql.DB, chatPassword *string, title *string) *ChatRoom {
	c := new(ChatRoom)
	c.subscribe = make(chan (chan<- Subscription), 10)
	c.unsubscribe = make(chan (<-chan Event), 10)
	c.publish = make(chan Event, 10)
	c.private = false
	c.protected = chatPassword != nil
	if c.protected {
		c.password = *chatPassword
	}
	c.title = "No name"
	if title != nil {
		c.title = *title
	}
	if db != nil {
		rows, err := db.Query("INSERT INTO rooms(password) VALUES($1) RETURNING id", chatPassword)
		fmt.Println(err)
		if err != nil {
			return nil
		}
		rows.Next()
		if err = rows.Scan(&c.id); err != nil {
			fmt.Println(err)
			return nil
		}
	}
	go c.Run()
	return c
}

// func NewPrivateChatRoom(user1 int, user2 int, db *sql.DB) *ChatRoom {
// 	c := new(ChatRoom)
// 	c.subscribe = make(chan (chan<- Subscription), 10)
// 	c.unsubscribe = make(chan (<-chan Event), 10)
// 	c.publish = make(chan Event, 10)
// 	c.private = true
// 	c.privateData.user1 = user1
// 	c.privateData.user2 = user2
// 	if db != nil {
// 		rows, err := db.Query("INSERT INTO private_rooms(user1, user2) VALUES () RETURNING id", user1, user2)
// 		if err != nil {
// 			return nil
// 		}
// 		if rows.Scan(c.id) == nil {
// 			return nil
// 		}
// 	}
// 	return c
// }

// func MakeChatRoom() ChatRoom {
// 	c := ChatRoom{}
// 	c.subscribe = make(chan (chan<- Subscription), 10)
// 	c.unsubscribe = make(chan (<-chan Event), 10)
// 	c.publish = make(chan Event, 10)
// 	return c
// }

type Event struct {
	Type      string
	Timestamp int
	Data      map[string]string
	User      struct {
		ID       int
		Username string
	}
}

type Subscription struct {
	// Archive []Event
	New <-chan Event
}

// func (s *Subscription) GetArchive(db *sql.DB) {
// 	rows, err := db.Query("SELECT id, author, date_created, text FROM messages WHERE room_id = $1 ORDER BY date_created", )
// }

func (chat *ChatRoom) Unsubscribe(s Subscription) {
	chat.unsubscribe <- s.New
	drain(s.New)
}

func newEvent(typ string, u *user.User, data map[string]string) Event {
	event := Event{}
	event.Type = typ
	event.Timestamp = int(time.Now().Unix())
	event.User.ID = u.ID
	event.User.Username = u.Username
	event.Data = data
	return event
}

func (chat *ChatRoom) Subscribe() Subscription {
	resp := make(chan Subscription)
	chat.subscribe <- resp
	return <-resp
}

func (chat *ChatRoom) Execute(env *Env, u *user.User, cmd Command) error {
	switch cmd.Action {
	case "message":
		msg, exists := cmd.Params["message"]
		if !exists {
			return nil
		}
		var err error
		_, err = env.DB.Query("INSERT INTO messages (room_id, author, text) VALUES ($1, $2, $3)", chat.id, u.ID, msg)
		if err != nil {
			return err
		}
		chat.Say(u, msg)
		return err
	case "delete":
		if u.ID == -1 {
			return nil
		}
		msg, exists := cmd.Params["message"]
		if !exists {
			return nil
		}
		msgID, err := strconv.Atoi(msg)
		if err != nil {
			return err
		}
		_, err = env.DB.Query("DELETE FROM messages WHERE = room_id = $1 AND id = $2 AND author = $3 RETURNING id", chat.id, msgID, u.ID)
		if err != nil {
			return err
		}
		chat.publish <- newEvent("delete", u, map[string]string{"deletedid": msg})
		return err
	case "alter":
		if u.ID == -1 {
			return nil
		}
		msg, exists := cmd.Params["message"]
		if !exists {
			return nil
		}
		text, exists := cmd.Params["text"]
		if !exists {
			return nil
		}
		msgID, err := strconv.Atoi(msg)
		if err != nil {
			return err
		}
		_, err = env.DB.Query("UPDATE private_messages SET text = $3 WHERE = room_id = $1 AND id = $2 AND author = $3 RETURNING id", chat.id, msgID, u.ID, text)
		if err != nil {
			return err
		}
		chat.publish <- newEvent("altered", u, map[string]string{"alteredid": msg, "message": text})
		return err
	case "resend":
		if u.ID == -1 {
			return nil
		}
		msg, exists := cmd.Params["message"]
		if !exists {
			return nil
		}
		msgID, err := strconv.Atoi(msg)
		if err != nil {
			return err
		}
		rows, err := env.DB.Query("INSERT INTO messages (room_id, author, text, resend) VALUES ($1, $2, (SELECT text FROM messages WHERE id = $3), true) RETURNIN text", chat.id, u.ID, msgID)
		if err != nil {
			return err
		}
		rows.Next()
		var text string
		err = rows.Scan(&text)
		if err != nil {
			return err
		}
		chat.publish <- newEvent("resend", u, map[string]string{"resendedid": msg, "message": text})
		return err
	case "reply":
		if u.ID == -1 {
			return nil
		}
		msg, exists := cmd.Params["message"]
		if !exists {
			return nil
		}
		text, exists := cmd.Params["text"]
		if !exists {
			return nil
		}
		msgID, err := strconv.Atoi(msg)
		if err != nil {
			return err
		}
		_, err = env.DB.Query("INSERT INTO messages (room_id, author, text, answerto) VALUES ($1, $2, $3, $4)", chat.id, u.ID, text, msgID)
		if err != nil {
			return err
		}
		chat.publish <- newEvent("reply", u, map[string]string{"replyto": strconv.Itoa(msgID), "message": text})
		return err
	}
	return nil
}

func (chat *ChatRoom) GetArchive(db *sql.DB) ([]historyEntry, error) {
	var rows *sql.Rows
	var err error
	rows, err = db.Query("SELECT id, author, date_created, text, resend, answerto FROM messages WHERE room_id = $1 ORDER BY date_created", chat.id)
	if err != nil {
		return nil, err
	}
	result := make([]historyEntry, 0)
	for rows.Next() {
		var entry historyEntry
		err = rows.Scan(&entry.ID, &entry.Author, &entry.DateCreated, &entry.Text, &entry.Resend, entry.Answerto)
		if err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, nil
}

func (chat *ChatRoom) Join(u *user.User) {
	chat.publish <- newEvent("join", u, map[string]string{})
	chat.users++
}

func (chat *ChatRoom) Say(u *user.User, message string) {
	chat.publish <- newEvent("message", u, map[string]string{"message": message})
}

func (chat *ChatRoom) Leave(u *user.User) {
	chat.publish <- newEvent("leave", u, map[string]string{})
	chat.users--
}

func (chat *ChatRoom) Run() {
	const archiveSize = 10
	// archive := list.New()
	subscribers := list.New()

	for {
		select {
		case ch := <-chat.subscribe:
			// var events []Event
			// for e := archive.Front(); e != nil; e = e.Next() {
			// 	events = append(events, e.Value.(Event))
			// }
			subscriber := make(chan Event, 10)
			subscribers.PushBack(subscriber)
			// ch <- Subscription{events, subscriber}
			ch <- Subscription{subscriber}

		case event := <-chat.publish:
			for ch := subscribers.Front(); ch != nil; ch = ch.Next() {
				ch.Value.(chan Event) <- event
			}
			// if archive.Len() >= archiveSize {
			// 	archive.Remove(archive.Front())
			// }
			// archive.PushBack(event)

		case unsub := <-chat.unsubscribe:
			for ch := subscribers.Front(); ch != nil; ch = ch.Next() {
				if ch.Value.(chan Event) == unsub {
					subscribers.Remove(ch)
					break
				}
			}
		}
	}
}

func drain(ch <-chan Event) {
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		default:
			return
		}
	}
}
