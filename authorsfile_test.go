package main

import "testing"

func TestAuthorsFile(t *testing.T) {
	cases := []struct {
		login string
		ok    bool
		exp   user
	}{
		{"foobar", true, user{"foobar", "Foo Bar", "foobar@example.com"}},
		{"bazquux", true, user{"bazquux", "Baz Quux", "baz@example.com"}},
		{"froba", true, user{"froba", "Frobble Banana, Jr.", "frobble.jr@example.com"}},
		{"other", false, user{}},
	}

	for _, tc := range cases {
		user, err := getUserFromFile(tc.login, "testdata/AUTHORS")
		if tc.ok {
			if err != nil {
				t.Error("Unexpected error", err)
			} else if user != tc.exp {
				t.Errorf("Unexpected user data, %v != expected %v", user, tc.exp)
			}
		} else if err == nil {
			t.Error("Unexpected nil error")
		}
	}
}
