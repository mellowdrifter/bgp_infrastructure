package main

import (
	"testing"
)

func TestDeltaMessage(t *testing.T) {
	var tests = []struct {
		name       string
		hour, week int
		output     string
	}{
		{
			name:   "test1",
			hour:   780710 - 780896,
			week:   780710 - 770567,
			output: "This is 186 fewer prefixes than 6 hours ago and 10143 more prefixes than a week ago",
		},
	}

	for _, test := range tests {
		actual := deltaMessage(test.hour, test.week)
		if actual != test.output {
			t.Errorf("Test %s output down not match. Wanted %s, received %s", test.name, test.output, actual)
		}
	}
}
