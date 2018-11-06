package handlers

import (
	"container/list"
	"time"
)

type ChatRoom struct {
	// Send a channel here to get room events back.  It will send the entire
	// archive initially, and then new messages as they come in.
	subscribe chan (chan<- Subscription)
	// Send a channel here to unsubscribe.
	unsubscribe chan (<-chan Event)
	// Send events here to publish them.
	publish chan Event
	users   int
}

func NewChatRoom() *ChatRoom {
	c := new(ChatRoom)
	c.subscribe = make(chan (chan<- Subscription), 10)
	c.unsubscribe = make(chan (<-chan Event), 10)
	c.publish = make(chan Event, 10)
	return c
}

func MakeChatRoom() ChatRoom {
	c := ChatRoom{}
	c.subscribe = make(chan (chan<- Subscription), 10)
	c.unsubscribe = make(chan (<-chan Event), 10)
	c.publish = make(chan Event, 10)
	return c
}

type Event struct {
	Type      string // "join", "leave", or "message"
	User      string
	Timestamp int    // Unix timestamp (secs)
	Text      string // What the user said (if Type == "message")
}

type Subscription struct {
	Archive []Event      // All the events from the archive.
	New     <-chan Event // New events coming in.
}

// Owner of a subscription must cancel it when they stop listening to events.
// func (s Subscription) Cancel() {
// 	unsubscribe <- s.New // Unsubscribe the channel.
// 	drain(s.New)         // Drain it, just in case there was a pending publish.
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

func (chat *ChatRoom) Join(user string) {
	chat.publish <- newEvent("join", user, "")
	chat.users++
}

func (chat *ChatRoom) Say(user, message string) {
	chat.publish <- newEvent("message", user, message)
}

func (chat *ChatRoom) Leave(user string) {
	chat.publish <- newEvent("leave", user, "")
	chat.users--
}

const archiveSize = 10

// This function loops forever, handling the chat room pubsub
func (chat *ChatRoom) Run() {
	archive := list.New()
	subscribers := list.New()

	for {
		select {
		case ch := <-chat.subscribe:
			var events []Event
			for e := archive.Front(); e != nil; e = e.Next() {
				events = append(events, e.Value.(Event))
			}
			subscriber := make(chan Event, 10)
			subscribers.PushBack(subscriber)
			ch <- Subscription{events, subscriber}

		case event := <-chat.publish:
			for ch := subscribers.Front(); ch != nil; ch = ch.Next() {
				ch.Value.(chan Event) <- event
			}
			if archive.Len() >= archiveSize {
				archive.Remove(archive.Front())
			}
			archive.PushBack(event)

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

// func init() {
// 	go chatroom()
// }

// Helpers

// Drains a given channel of any messages.
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
