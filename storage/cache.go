package storage

type Cache map[string]map[string]bool

func (c Cache) Clear(k string) {
	delete(c, k)
}

func (c Cache) Set(k, v string) {
	sm := c[k]
	if sm == nil {
		sm = map[string]bool{}
	}
	sm[v] = true
	c[k] = sm
}

func (c Cache) Has(k, v string) bool {
	sm := c[k]
	if sm == nil {
		return false
	}
	return sm[v]
}
