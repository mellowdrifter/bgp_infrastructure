package main

import (
	"reflect"
	"testing"
	"time"
)

func TestDeltaMessage(t *testing.T) {
	tests := []struct {
		name       string
		hour, week int
		output     string
	}{
		{
			name:   "test1",
			hour:   780710 - 780896,
			week:   780710 - 770567,
			output: "This is 186 fewer prefixes than 6 hours ago and 10143 more than a week ago",
		},
	}

	for _, test := range tests {
		actual := deltaMessage(test.hour, test.week)
		if actual != test.output {
			t.Errorf("Test %s output down not match. Wanted %q, received %q", test.name, test.output, actual)
		}
	}
}

func TestWhatToTweet(t *testing.T) {
	tests := []struct {
		name string
		time string
		want toTweet
	}{
		{
			name: "Midnight",
			time: "2006-01-01T00:00:00Z",
			want: toTweet{},
		},
		{
			name: "Monday, 20:00",
			time: "2020-01-06T20:00:00Z",
			want: toTweet{
				annualGraph: true,
				tableSize:   true,
				weekGraph:   true,
			},
		},
		{
			name: "Tuesday, 20:00",
			time: "2020-01-21T20:00:00Z",
			want: toTweet{
				tableSize: true,
			},
		},
		{
			name: "Wednesday, 20:00",
			time: "2020-01-08T20:30:00Z",
			want: toTweet{
				tableSize: true,
				subnetPie: true,
			},
		},
		{
			name: "Thursday, 20:00",
			time: "2020-01-30T20:14:57Z",
			want: toTweet{
				tableSize: true,
				rpkiPie:   true,
			},
		},
		{
			name: "Friday, 20:00",
			time: "2020-01-03T20:00:00Z",
			want: toTweet{
				tableSize: true,
			},
		},
		{
			name: "Monday, 20:00, first day of month",
			time: "2020-02-03T20:00:00Z",
			want: toTweet{
				tableSize:  true,
				weekGraph:  true,
				monthGraph: true,
			},
		},
		{
			name: "Wednesday, 20:00, first day of July",
			time: "2020-07-01T20:00:00Z",
			want: toTweet{
				tableSize:     true,
				monthGraph:    true,
				sixMonthGraph: true,
				subnetPie:     true,
			},
		},
		{
			name: "Saturday, 20:00, first day of July 2023",
			time: "2023-07-01T20:00:00Z",
			want: toTweet{
				tableSize: true,
			},
		},
		{
			name: "Sunday, 20:00, second day of July 2023",
			time: "2023-07-02T20:00:00Z",
			want: toTweet{
				tableSize: true,
			},
		},
		{
			name: "Monday, 20:00, third day of July 2023",
			time: "2023-07-03T20:00:00Z",
			want: toTweet{
				tableSize:     true,
				weekGraph:     true,
				monthGraph:    true,
				sixMonthGraph: true,
			},
		},
		{
			name: "Monday, 21:00, third day of July 2023",
			time: "2023-07-03T21:00:00Z",
			want: toTweet{},
		},
	}

	for _, tt := range tests {
		time, err := time.Parse(time.RFC3339, tt.time)
		if err != nil {
			t.Errorf("unable to parse time: %s (%v)", tt.time, err)
		}
		got := whatToTweet(time)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s failed. got %#v, want %#v", tt.name, got, tt.want)
		}
	}
}
