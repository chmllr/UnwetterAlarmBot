package data

import "fmt"

type Volume map[string][]int

func (db Volume) Register(userID int, plz string) error {
	subscribers := db[plz]
	for _, v := range subscribers {
		if v == userID {
			return fmt.Errorf("user %d is already subscribed to PLZ %q", userID, plz)
		}
	}
	db[plz] = append(subscribers, userID)
	return nil
}

func (db Volume) Unregister(userID int) int {
	plzs := 0
L:
	for plz, subscribers := range db {
		for i, v := range subscribers {
			if v == userID {
				db[plz] = append(subscribers[0:i], subscribers[i+1:]...)
				plzs++
				continue L
			}
		}
	}
	return plzs
}

func (db Volume) subscribers(plz string) []int {
	return db[plz]
}

func (db Volume) PLZs() (plzs []string) {
	for plz, subscribers := range db {
		if len(subscribers) == 0 {
			continue
		}
		plzs = append(plzs, plz)
	}
	return
}
