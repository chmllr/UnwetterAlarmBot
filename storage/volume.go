package storage

import (
	"fmt"
	"sort"
	"sync"
)

type Subscriber struct {
	UserID int
	ChatID int64
}
type Volume struct {
	db    map[string][]Subscriber
	mutex sync.Mutex
}

func (v *Volume) Load(file string) {
	v.db = map[string][]Subscriber{}
}

func (v *Volume) Register(userID int, chatID int64, plz string) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	subscribers := v.db[plz]
	for _, v := range subscribers {
		if v.UserID == userID {
			return fmt.Errorf("user %d is already subscribed to PLZ %q", userID, plz)
		}
	}
	v.db[plz] = append(subscribers, Subscriber{userID, chatID})
	return nil
}

func (v *Volume) Unregister(userID int) int {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	plzs := 0
L:
	for plz, subscribers := range v.db {
		for i, s := range subscribers {
			if s.UserID == userID {
				v.db[plz] = append(subscribers[0:i], subscribers[i+1:]...)
				plzs++
				continue L
			}
		}
	}
	return plzs
}

func (v *Volume) Subscribers(plz string) []Subscriber {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	return v.db[plz]
}

func (v *Volume) PLZs() (plzs []string) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	for plz, subscribers := range v.db {
		if len(subscribers) == 0 {
			continue
		}
		plzs = append(plzs, plz)
	}
	sort.Strings(plzs)
	return
}
