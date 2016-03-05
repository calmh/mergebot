package main

import "testing"

func TestReflow(t *testing.T) {
	cases := [][2]string{
		{"foo", "foo\n"},
		{"foo bar baz quux", "foo bar\nbaz quux\n"},
		{"foo bar\nbaz quux", "foo bar\nbaz quux\n"},
		{"foo bar\n  baz quux", "foo bar\n\n  baz quux\n"},
		{"foo bar\n  baz quux\n  baz baz", "foo bar\n\n  baz quux\n  baz baz\n"},
	}

	for _, tc := range cases {
		actual := reflow(tc[0], 8)
		if actual != tc[1] {
			t.Errorf("Reflowed %q into %q, expected %q", tc[0], actual, tc[1])
		}
	}
}
