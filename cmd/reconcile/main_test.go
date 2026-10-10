package main

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
	"google.golang.org/grpc"
)

type mockReconcileServer struct {
	pb.UnimplementedBgpInfoServer
	samples map[string]*pb.Values
}

func (m *mockReconcileServer) GetSampleIndex(_ context.Context, req *pb.SampleIndexRequest) (*pb.SampleIndexResponse, error) {
	var keys []*pb.SampleKey
	for _, s := range m.samples {
		if s.GetTime() >= req.GetSinceTime() {
			keys = append(keys, &pb.SampleKey{
				Source:  s.GetSource(),
				Time:    s.GetTime(),
				Quality: s.GetQuality(),
				Tweeted: s.GetTweeted(),
			})
		}
	}
	return &pb.SampleIndexResponse{Keys: keys}, nil
}

func (m *mockReconcileServer) UpdateTweetBit(_ context.Context, req *pb.Timestamp) (*pb.Result, error) {
	for _, s := range m.samples {
		if s.GetTime() == req.GetTime() {
			s.Tweeted = true
		}
	}
	return &pb.Result{Success: true}, nil
}

func (m *mockReconcileServer) GetSampleBatch(_ context.Context, req *pb.SampleBatchRequest) (*pb.SampleBatchResponse, error) {
	var out []*pb.Values
	for _, k := range req.GetKeys() {
		for _, s := range m.samples {
			if s.GetSource() == k.GetSource() && s.GetTime() == k.GetTime() {
				out = append(out, s)
			}
		}
	}
	return &pb.SampleBatchResponse{Samples: out}, nil
}

func (m *mockReconcileServer) AddSampleBatch(_ context.Context, req *pb.SampleBatchResponse) (*pb.Result, error) {
	for _, s := range req.GetSamples() {
		m.samples[fmt.Sprintf("%s@%d", s.GetSource(), s.GetTime())] = s
	}
	return &pb.Result{Success: true}, nil
}

func startMockServer(t *testing.T, initialSamples []*pb.Values) (*mockReconcileServer, string, func()) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	mock := &mockReconcileServer{samples: make(map[string]*pb.Values)}
	for _, s := range initialSamples {
		mock.samples[fmt.Sprintf("%s@%d", s.GetSource(), s.GetTime())] = s
	}
	pb.RegisterBgpInfoServer(srv, mock)

	go srv.Serve(lis)

	cleanup := func() {
		srv.Stop()
		lis.Close()
	}
	return mock, lis.Addr().String(), cleanup
}

func TestRunReconciliation(t *testing.T) {
	now := uint64(time.Now().Unix())
	now = now - (now % 300)

	s1Local := &pb.Values{Source: "bgp1", Time: now - 600, Quality: "ok", Tweeted: true}
	s1Remote := &pb.Values{Source: "bgp1", Time: now - 600, Quality: "ok", Tweeted: false}
	s2 := &pb.Values{Source: "bgp3", Time: now - 600, Quality: "ok", Tweeted: false}
	s3 := &pb.Values{Source: "bgp1", Time: now - 300, Quality: "suspect", Tweeted: false}

	// Local has s1Local (tweeted), s2
	_, localAddr, stopLocal := startMockServer(t, []*pb.Values{s1Local, s2})
	defer stopLocal()

	// Remote has s1Remote (not tweeted), s3
	mockRemote, remoteAddr, stopRemote := startMockServer(t, []*pb.Values{s1Remote, s3})
	defer stopRemote()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Dry run
	stats, err := runReconciliation(ctx, localAddr, remoteAddr, 1, 10, true)
	if err != nil {
		t.Fatalf("Dry run failed: %v", err)
	}
	if stats.localCount != 2 || stats.remoteCount != 2 {
		t.Errorf("Expected 2 local and 2 remote keys, got local=%d, remote=%d", stats.localCount, stats.remoteCount)
	}
	if stats.copiedToLocal != 0 || stats.copiedToRemote != 0 {
		t.Errorf("Expected 0 copied in dry run, got local=%d, remote=%d", stats.copiedToLocal, stats.copiedToRemote)
	}
	if stats.tweetBitsToRemote != 1 {
		t.Errorf("Expected 1 tweet bit to sync to remote in dry run, got %d", stats.tweetBitsToRemote)
	}
	if stats.tweetBitsSyncedToRemote != 0 {
		t.Errorf("Expected 0 tweet bits synced in dry run, got %d", stats.tweetBitsSyncedToRemote)
	}

	// 2. Real reconciliation
	stats, err = runReconciliation(ctx, localAddr, remoteAddr, 1, 10, false)
	if err != nil {
		t.Fatalf("Reconciliation failed: %v", err)
	}
	if stats.copiedToLocal != 1 {
		t.Errorf("Expected 1 copied to local (s3), got %d", stats.copiedToLocal)
	}
	if stats.copiedToRemote != 1 {
		t.Errorf("Expected 1 copied to remote (s2), got %d", stats.copiedToRemote)
	}
	if stats.tweetBitsSyncedToRemote != 1 {
		t.Errorf("Expected 1 tweet bit synced to remote, got %d", stats.tweetBitsSyncedToRemote)
	}
	if !mockRemote.samples[fmt.Sprintf("bgp1@%d", now-600)].GetTweeted() {
		t.Errorf("Expected remote sample bgp1@%d to have Tweeted=true after reconciliation", now-600)
	}
}
