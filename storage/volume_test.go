package storage

import (
	"reflect"
	"testing"
)

func Test_Volume(t *testing.T) {

	vol := Volume{}

	user_1 := 1
	user_2 := 2
	user_3 := 3
	plz_5621 := "5621"
	plz_8045 := "8045"

	err := vol.Register(user_1, 11, plz_5621)
	if err != nil {
		t.Fatalf("couldn't register user %d for PLZ %q", user_1, plz_5621)
	}

	err = vol.Register(user_1, 11, plz_8045)
	if err != nil {
		t.Fatalf("couldn't register user %d for PLZ %q", user_1, plz_8045)
	}

	plzsWant := []string{"5621", "8045"}
	plzsGot := vol.PLZs()
	if !reflect.DeepEqual(plzsGot, plzsWant) {
		t.Fatalf("PLZs wanted %v, got %v", plzsWant, plzsGot)
	}

	plzs := vol.Unregister(user_2)
	wantPlzs := 0
	if plzs != wantPlzs {
		t.Fatalf("user %d should have been unregistered for %d plzs, but got %d", user_2, wantPlzs, plzs)
	}

	exp := []Subscriber{{1, 11}}
	got := vol.Subscribers(plz_5621)
	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("subscribers for PLZ %q expected %v, but got %v", plz_5621, exp, got)
	}

	plzs = vol.Unregister(user_1)
	wantPlzs = 2
	if plzs != wantPlzs {
		t.Fatalf("user %d should have been unregistered for %d plzs, but got %d", user_1, wantPlzs, plzs)
	}

	exp = []Subscriber{}
	got = vol.Subscribers(plz_5621)
	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("subscribers for PLZ %q expected %v, but got %v", plz_5621, exp, got)
	}

	vol.Register(user_1, 11, plz_5621)
	vol.Register(user_2, 22, plz_5621)

	exp = []Subscriber{{1, 11}, {2, 22}}
	got = vol.Subscribers(plz_5621)
	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("subscribers for PLZ %q expected %v, but got %v", plz_5621, exp, got)
	}

	vol.Register(user_3, 33, plz_5621)
	exp = []Subscriber{{1, 11}, {2, 22}, {3, 33}}
	got = vol.Subscribers(plz_5621)
	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("subscribers for PLZ %q expected %v, but got %v", plz_5621, exp, got)
	}

	vol.Unregister(user_2)
	exp = []Subscriber{{1, 11}, {3, 33}}
	got = vol.Subscribers(plz_5621)
	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("subscribers for PLZ %q expected %v, but got %v", plz_5621, exp, got)
	}

	vol.Unregister(user_1)
	vol.Unregister(user_3)
	exp = []Subscriber{}
	got = vol.Subscribers(plz_5621)
	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("subscribers for PLZ %q expected %v, but got %v", plz_5621, exp, got)
	}

	plzsWant = nil
	plzsGot = vol.PLZs()
	if !reflect.DeepEqual(plzsGot, plzsWant) {
		t.Fatalf("PLZs wanted %v, got %v", plzsWant, plzsGot)
	}

}
