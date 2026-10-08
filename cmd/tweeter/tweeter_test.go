package main

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	bpb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
	"google.golang.org/grpc"
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
			output: "This is 186 fewer prefixes than 6 hours ago and 10143 more than a week ago.",
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
			want: toTweet{test: true},
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
			want: toTweet{test: true},
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

type mockBgpClient struct {
	resp *bpb.PrefixCountResponse
	err  error
}

func (m *mockBgpClient) GetPrefixCount(_ context.Context, _ *bpb.Empty, _ ...grpc.CallOption) (*bpb.PrefixCountResponse, error) {
	return m.resp, m.err
}
func (m *mockBgpClient) GetPieSubnets(_ context.Context, _ *bpb.Empty, _ ...grpc.CallOption) (*bpb.PieSubnetsResponse, error) {
	return nil, nil
}
func (m *mockBgpClient) GetMovementTotals(_ context.Context, _ *bpb.MovementRequest, _ ...grpc.CallOption) (*bpb.MovementTotalsResponse, error) {
	return nil, nil
}
func (m *mockBgpClient) GetRpki(_ context.Context, _ *bpb.Empty, _ ...grpc.CallOption) (*bpb.Roas, error) {
	return nil, nil
}
func (m *mockBgpClient) UpdateTweetBit(_ context.Context, _ *bpb.Timestamp, _ ...grpc.CallOption) (*bpb.Result, error) {
	return nil, nil
}
func (m *mockBgpClient) UpdateAsnames(_ context.Context, _ *bpb.AsnamesRequest, _ ...grpc.CallOption) (*bpb.Result, error) {
	return nil, nil
}
func (m *mockBgpClient) AddLatest(_ context.Context, _ *bpb.Values, _ ...grpc.CallOption) (*bpb.Result, error) {
	return nil, nil
}
func (m *mockBgpClient) AddValues(_ context.Context, _ *bpb.Values, _ ...grpc.CallOption) (*bpb.Result, error) {
	return nil, nil
}
func (m *mockBgpClient) GetAsname(_ context.Context, _ *bpb.GetAsnameRequest, _ ...grpc.CallOption) (*bpb.GetAsnameResponse, error) {
	return nil, nil
}
func (m *mockBgpClient) GetAsnames(_ context.Context, _ *bpb.Empty, _ ...grpc.CallOption) (*bpb.GetAsnamesResponse, error) {
	return nil, nil
}
func (m *mockBgpClient) GetSampleIndex(_ context.Context, _ *bpb.SampleIndexRequest, _ ...grpc.CallOption) (*bpb.SampleIndexResponse, error) {
	return nil, nil
}
func (m *mockBgpClient) GetSampleBatch(_ context.Context, _ *bpb.SampleBatchRequest, _ ...grpc.CallOption) (*bpb.SampleBatchResponse, error) {
	return nil, nil
}
func (m *mockBgpClient) AddSampleBatch(_ context.Context, _ *bpb.SampleBatchResponse, _ ...grpc.CallOption) (*bpb.Result, error) {
	return nil, nil
}

func TestStalenessGuard(t *testing.T) {
	now := time.Now().Unix()

	// 1. Sample older than 15 minutes (e.g. 20 minutes old)
	mockStale := &mockBgpClient{
		resp: &bpb.PrefixCountResponse{
			Time:     uint64(now - 1200),
			Active_4: 1000000,
			Active_6: 250000,
		},
	}
	_, err := current(mockStale, true)
	if err == nil || !strings.Contains(err.Error(), "exceeds 15m limit") {
		t.Errorf("Expected 15m staleness error for 20m old sample, got: %v", err)
	}

	// 2. Sample timestamp zero
	mockZero := &mockBgpClient{
		resp: &bpb.PrefixCountResponse{
			Time:     0,
			Active_4: 1000000,
			Active_6: 250000,
		},
	}
	_, err = current(mockZero, true)
	if err == nil || !strings.Contains(err.Error(), "missing or zero") {
		t.Errorf("Expected missing/zero timestamp error, got: %v", err)
	}

	// 3. Fresh sample (2 minutes old)
	mockFresh := &mockBgpClient{
		resp: &bpb.PrefixCountResponse{
			Time:        uint64(now - 120),
			Active_4:    1000000,
			Active_6:    250000,
			Sixhoursv4:  1000000,
			Sixhoursv6:  250000,
			Weekagov4:   990000,
			Weekagov6:   240000,
		},
	}
	tweets, err := current(mockFresh, true)
	if err != nil {
		t.Errorf("Expected fresh sample to succeed, got error: %v", err)
	}
	if len(tweets) == 0 {
		t.Errorf("Expected tweets to be returned for fresh sample")
	}
}

