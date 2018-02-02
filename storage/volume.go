package storage

import (
	"fmt"
	"sort"
)

type Subscriber struct {
	UserID int
	ChatID int64
}
type Volume map[string][]Subscriber

func (db Volume) Register(userID int, chatID int64, plz string) error {
	subscribers := db[plz]
	for _, v := range subscribers {
		if v.UserID == userID {
			return fmt.Errorf("user %d is already subscribed to PLZ %q", userID, plz)
		}
	}
	db[plz] = append(subscribers, Subscriber{userID, chatID})
	return nil
}

func (db Volume) Unregister(userID int) int {
	plzs := 0
L:
	for plz, subscribers := range db {
		for i, v := range subscribers {
			if v.UserID == userID {
				db[plz] = append(subscribers[0:i], subscribers[i+1:]...)
				plzs++
				continue L
			}
		}
	}
	return plzs
}

func (db Volume) Subscribers(plz string) []Subscriber {
	return db[plz]
}

func (db Volume) PLZs() (plzs []string) {
	for plz, subscribers := range db {
		if len(subscribers) == 0 {
			continue
		}
		plzs = append(plzs, plz)
	}
	sort.Strings(plzs)
	return
}
