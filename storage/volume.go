package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
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

var volFile string

func (v *Volume) Load(path string) error {
	v.db = map[string][]Subscriber{}
	volFile = path
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	data, err := ioutil.ReadFile(volFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &v.db)
}

func (v *Volume) Persist() error {
	if volFile == "" {
		panic("no file for volume persistence specified")
	}
	data, err := json.Marshal(v.db)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(volFile, data, 0644)
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
	return v.Persist()
}

func (v *Volume) Unregister(userID int) (int, error) {
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
	return plzs, v.Persist()
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
