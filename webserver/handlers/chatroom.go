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

func NewPrivateChatRoom(user1 int, user2 int, db *sql.DB) *ChatRoom {
	c := new(ChatRoom)
	c.subscribe = make(chan (chan<- Subscription), 10)
	c.unsubscribe = make(chan (<-chan Event), 10)
	c.publish = make(chan Event, 10)
	c.private = true
	c.privateData.user1 = user1
	c.privateData.user2 = user2
	if db != nil {
		rows, err := db.Query("INSERT INTO private_rooms(user1, user2) VALUES () RETURNING id", user1, user2)
		if err != nil {
			return nil
		}
		if rows.Scan(c.id) == nil {
			return nil
		}
	}
	return c
}

// func MakeChatRoom() ChatRoom {
// 	c := ChatRoom{}
// 	c.subscribe = make(chan (chan<- Subscription), 10)
// 	c.unsubscribe = make(chan (<-chan Event), 10)
// 	c.publish = make(chan Event, 10)
// 	return c
// }

type Event struct {
	Type      string
	User      string
	Timestamp int
	Text      string
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

func newEvent(typ, user, msg string) Event {
	return Event{typ, user, int(time.Now().Unix()), msg}
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
		if chat.private {
			_, err = env.DB.Query("INSERT INTO private_messages (room_id, author, text) VALUES ($1, $2, $3)", chat.id, u.ID, msg)
		} else {
			_, err = env.DB.Query("INSERT INTO messages (room_id, author, text) VALUES ($1, $2, $3)", chat.id, u.ID, msg)
		}
		if err != nil {
			return err
		}
		chat.Say(u, msg)
		return err
	}
	return nil
}

func (chat *ChatRoom) GetArchive(db *sql.DB) ([]historyEntry, error) {
	var rows *sql.Rows
	var err error
	if chat.private {
		rows, err = db.Query("SELECT id, author, date_created, text FROM private_messages WHERE room_id = $1 ORDER BY date_created", chat.id)
	} else {
		rows, err = db.Query("SELECT id, author, date_created, text FROM messages WHERE room_id = $1 ORDER BY date_created", chat.id)
	}
	if err != nil {
		return nil, err
	}
	result := make([]historyEntry, 0)
	for rows.Next() {
		var entry historyEntry
		err = rows.Scan(&entry.ID, &entry.Author, &entry.DateCreated, &entry.Text)
		if err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, nil
}

func (chat *ChatRoom) Join(u *user.User) {
	chat.publish <- newEvent("join", strconv.Itoa(u.ID), "")
	chat.users++
}

func (chat *ChatRoom) Say(u *user.User, message string) {
	chat.publish <- newEvent("message", strconv.Itoa(u.ID), message)
}

func (chat *ChatRoom) Leave(u *user.User) {
	chat.publish <- newEvent("leave", strconv.Itoa(u.ID), "")
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
