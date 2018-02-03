package storage

import (
	"testing"
)

func TestCache(t *testing.T) {
	c := Cache{}
	c.Set("5621", "xxx")
	c.Set("5621", "yyy")
	c.Set("8045", "zzz")

	for i, v := range []struct {
		plz, hash string
		has       bool
	}{
		{"5621", "xxx", true},
		{"5621", "yyy", true},
		{"5621", "zzz", false},
		{"8045", "xxx", false},
		{"8045", "zzz", true},
	} {
		if c.Has(v.plz, v.hash) != v.has {
			t.Fatalf("%d: %q, %q presense in cache is not as expected", i, v.plz, v.hash)
		}
	}

	c.Clear("5621")
	c.Clear("8045")

	if len(c) != 0 {
		t.Fatal("cache clearance failed")
	}
}
