package main

import (
	"reflect"
	"testing"
)

func TestFieldValues(t *testing.T) {
	checks := []struct {
		in    string
		field string
		out   []string
	}{
		{"foo bar baz", "quux", nil},
		{"foo\bar\nbaz\nfoo: bar baz\nquux", "quux", nil},
		{"foo\bar\nbaz\nfoo: bar baz\nquux", "foo", []string{"bar", "baz"}},
		{"foo\bar\nbaz\nFOo: bar baz\nfOO: quux\nquux", "foo", []string{"bar", "baz", "quux"}},
	}

	for _, tc := range checks {
		out := fieldValues(tc.in, tc.field)
		if !reflect.DeepEqual(out, tc.out) {
			t.Error("Got", out, "expected", tc.out)
		}
	}
}
