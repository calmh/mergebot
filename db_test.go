package main

import (
	"os"
	"reflect"
	"testing"
)

func TestLGTMPersistence(t *testing.T) {
	os.RemoveAll("_db")
	defer os.RemoveAll("_db")
	db, err := OpenDB("_db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	lgtms := db.LGTMs(1234)
	expected := []string(nil)
	if !reflect.DeepEqual(lgtms, expected) {
		t.Errorf("%+v != %+v", lgtms, expected)
	}

	db.LGTM(1234, "jb")
	db.LGTM(1234, "ab")

	lgtms = db.LGTMs(1234)
	expected = []string{"jb", "ab"}
	if !reflect.DeepEqual(lgtms, expected) {
		t.Errorf("%+v != %+v", lgtms, expected)
	}
}
