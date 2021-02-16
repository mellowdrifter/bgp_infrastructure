package main

import (
	"reflect"
	"testing"
)

func TestLoadAirports(t *testing.T) {
	t.Parallel()
	airFile := "/home/mellowd/go/src/github.com/mellowdrifter/bgp_infrastructure/glass/airports/airports.dat"
	airports, err := loadAirports(airFile)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		airport string
		want    location
	}{
		{
			airport: "CPT",
			want: location{
				city:    "Cape Town",
				country: "South Africa",
				lat:     "-33.9648017883",
				long:    "18.6016998291",
			},
		},
		{
			airport: "SIN",
			want: location{
				city:    "Singapore",
				country: "Singapore",
				lat:     "1.35019",
				long:    "103.994003",
			},
		},
		{
			airport: "HND",
			want: location{
				city:    "Tokyo",
				country: "Japan",
				lat:     "35.552299",
				long:    "139.779999",
			},
		},
		{
			airport: "HEL",
			want: location{
				city:    "Helsinki",
				country: "Finland",
				lat:     "60.317199707031",
				long:    "24.963300704956",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.airport, func(t *testing.T) {
			got, ok := airports[tc.airport]
			if !ok {
				t.Errorf("Airport should be there, but not")
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got: %v, want: %v", got, tc.want)
			}
		})
	}
}
