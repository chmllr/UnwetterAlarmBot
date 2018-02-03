package storage

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

var cacheFile string

type Cache map[string]map[string]bool

func (c Cache) Load(path string) error {
	cacheFile = path
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &c)
}

func (c Cache) Persist() error {
	if cacheFile == "" {
		panic("no file for cache persistence specified")
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(cacheFile, data, 0644)
}

func (c Cache) Clear(k string) {
	delete(c, k)
	defer c.Persist()
}

func (c Cache) Set(k, v string) {
	sm := c[k]
	if sm == nil {
		sm = map[string]bool{}
	}
	sm[v] = true
	c[k] = sm
	defer c.Persist()
}

func (c Cache) Has(k, v string) bool {
	sm := c[k]
	if sm == nil {
		return false
	}
	return sm[v]
}
