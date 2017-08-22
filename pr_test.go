package main

import (
	"testing"
)

func TestOverallStatus(t *testing.T) {
	cases := []struct {
		statuses []status
		skip     []string
		req      []string
		overall  prState
	}{
		{
			statuses: []status{{Context: "t1", State: statePending}},
			overall:  statePending,
		},
		{
			statuses: []status{{Context: "t1", State: stateSuccess}},
			overall:  stateSuccess,
		},
		{
			statuses: []status{{Context: "t1", State: stateError}},
			overall:  stateError,
		},
		{
			statuses: []status{{Context: "t1", State: statePending}, {Context: "t2", State: statePending}},
			overall:  statePending,
		},
		{
			statuses: []status{{Context: "t1", State: stateSuccess}, {Context: "t2", State: statePending}},
			overall:  statePending,
		},
		{
			statuses: []status{{Context: "t1", State: stateError}, {Context: "t2", State: statePending}},
			overall:  stateError,
		},
		{
			statuses: []status{{Context: "t1", State: stateError}, {Context: "t2", State: statePending}, {Context: "t2", State: stateSuccess}},
			overall:  stateError,
		},
		{
			statuses: []status{{Context: "t1", State: stateError}, {Context: "t2", State: statePending}, {Context: "t3", State: stateSuccess}},
			skip:     []string{"t1"},
			overall:  statePending,
		},
		{
			statuses: []status{{Context: "t1", State: stateError}, {Context: "t2", State: statePending}, {Context: "t3", State: stateSuccess}},
			skip:     []string{"t1", "t2"},
			overall:  stateSuccess,
		},
		{
			statuses: []status{{Context: "t1", State: stateError}, {Context: "t2", State: statePending}, {Context: "t3", State: stateSuccess}},
			skip:     []string{"t1", "t2"},
			req:      []string{"t1", "t2", "t3"},
			overall:  stateSuccess,
		},
		{
			statuses: []status{{Context: "t1", State: stateError}, {Context: "t2", State: statePending}, {Context: "t3", State: stateSuccess}},
			skip:     []string{"t1", "t2"},
			req:      []string{"t1", "t2", "t3", "t4"},
			overall:  statePending,
		},
	}

	for i, tc := range cases {
		res := overallStatus(tc.statuses, tc.skip, tc.req)
		if res != tc.overall {
			t.Errorf("%d: got %v, expected %v", i, res, tc.overall)
		}
	}
}
