package main

import "testing"

func TestParseBody(t *testing.T) {
	cases := []struct {
		b string
		p body
	}{
		{"", body{}},
		{"foo bar", body{command: "foo bar"}},
		{"@st-review: foo bar", body{recipient: "st-review", command: "foo bar"}},
		{"@st-review foo bar", body{recipient: "st-review", command: "foo bar"}},
		{" @st-review:  foo  bar ", body{recipient: "st-review", command: "foo bar"}},
		{" @st-review:  foo  bar \nSubject here\nBody here\nMore body", body{recipient: "st-review", command: "foo bar", subject: "Subject here", description: "Body here\nMore body"}},
		{" @st-review:  foo  bar \n\nSubject here\n\nBody here\nMore body\n\nMore", body{recipient: "st-review", command: "foo bar", subject: "Subject here", description: "Body here\nMore body\n\nMore"}},
	}

	for _, tc := range cases {
		actual := parseBody(tc.b)
		if actual != tc.p {
			t.Errorf("Expected %q to parse into %#v, not %#v", tc.b, tc.p, actual)
		}
	}
}
