package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type reconcileStats struct {
	windowStart     time.Time
	windowEnd       time.Time
	localCount      int
	remoteCount     int
	localSuspect    int
	remoteSuspect   int
	copiedToLocal   int
	copiedToRemote  int
	remainingGaps   map[string]int
	dryRun          bool
}

func main() {
	days := flag.Int("days", 35, "Number of days of history to reconcile")
	localAddr := flag.String("local-addr", "127.0.0.1:7179", "Local bgpsql address")
	remoteAddr := flag.String("remote-addr", "100.68.70.100:7179", "Remote bgpsql address")
	batchSize := flag.Int("batch-size", 100, "Batch size for sample transfer")
	dryRun := flag.Bool("dry-run", false, "Only audit differences without copying rows")
	healthcheckURL := flag.String("healthcheck-url", "", "Optional Healthchecks.io ping URL on success")
	timeoutSec := flag.Int("timeout", 600, "Overall timeout in seconds")
	flag.Parse()

	log.Printf("Starting BGP reconcile (window=%d days, local=%s, remote=%s, dry-run=%v)",
		*days, *localAddr, *remoteAddr, *dryRun)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeoutSec)*time.Second)
	defer cancel()

	stats, err := runReconciliation(ctx, *localAddr, *remoteAddr, *days, *batchSize, *dryRun)
	if err != nil {
		log.Fatalf("Reconciliation failed: %v", err)
	}

	printReport(stats, *localAddr, *remoteAddr)

	if *healthcheckURL != "" && !*dryRun {
		pingHealthcheck(*healthcheckURL)
	}
}

func runReconciliation(ctx context.Context, localAddr, remoteAddr string, days, batchSize int, dryRun bool) (*reconcileStats, error) {
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(64*1024*1024),
			grpc.MaxCallSendMsgSize(64*1024*1024),
		),
	}

	// Connect to local bgpsql
	localConn, err := grpc.DialContext(ctx, localAddr, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial local bgpsql (%s): %w", localAddr, err)
	}
	defer localConn.Close()
	localClient := pb.NewBgpInfoClient(localConn)

	// Connect to remote bgpsql
	remoteConn, err := grpc.DialContext(ctx, remoteAddr, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial remote bgpsql (%s): %w", remoteAddr, err)
	}
	defer remoteConn.Close()
	remoteClient := pb.NewBgpInfoClient(remoteConn)

	now := time.Now()
	var sinceTime uint64
	var windowStartTime time.Time
	if days > 0 {
		startTime := now.Add(-time.Duration(days) * 24 * time.Hour)
		sinceTime = uint64(startTime.Unix() - (startTime.Unix() % 300))
		windowStartTime = time.Unix(int64(sinceTime), 0).UTC()
	} else {
		sinceTime = 0
		windowStartTime = time.Unix(1418915400, 0).UTC() // Earliest sample: 2014-12-18
	}
	untilTime := uint64(now.Unix())

	stats := &reconcileStats{
		windowStart:   windowStartTime,
		windowEnd:     now.UTC(),
		remainingGaps: make(map[string]int),
		dryRun:        dryRun,
	}

	// 1. Fetch sample indices
	log.Printf("Fetching sample index from local (%s) since %s...", localAddr, stats.windowStart)
	localIdx, err := localClient.GetSampleIndex(ctx, &pb.SampleIndexRequest{SinceTime: sinceTime, UntilTime: untilTime})
	if err != nil {
		return nil, fmt.Errorf("GetSampleIndex on local failed: %w", err)
	}

	log.Printf("Fetching sample index from remote (%s) since %s...", remoteAddr, stats.windowStart)
	remoteIdx, err := remoteClient.GetSampleIndex(ctx, &pb.SampleIndexRequest{SinceTime: sinceTime, UntilTime: untilTime})
	if err != nil {
		return nil, fmt.Errorf("GetSampleIndex on remote failed: %w", err)
	}

	stats.localCount = len(localIdx.GetKeys())
	stats.remoteCount = len(remoteIdx.GetKeys())

	localMap := make(map[string]*pb.SampleKey, stats.localCount)
	for _, k := range localIdx.GetKeys() {
		key := fmt.Sprintf("%s@%d", k.GetSource(), k.GetTime())
		localMap[key] = k
		if k.GetQuality() == "suspect" {
			stats.localSuspect++
		}
	}

	remoteMap := make(map[string]*pb.SampleKey, stats.remoteCount)
	for _, k := range remoteIdx.GetKeys() {
		key := fmt.Sprintf("%s@%d", k.GetSource(), k.GetTime())
		remoteMap[key] = k
		if k.GetQuality() == "suspect" {
			stats.remoteSuspect++
		}
	}

	// 2. Compute missing keys
	var missingOnLocal []*pb.SampleKey
	for key, k := range remoteMap {
		if _, ok := localMap[key]; !ok {
			missingOnLocal = append(missingOnLocal, k)
		}
	}

	var missingOnRemote []*pb.SampleKey
	for key, k := range localMap {
		if _, ok := remoteMap[key]; !ok {
			missingOnRemote = append(missingOnRemote, k)
		}
	}

	// Sort missing keys chronologically
	sort.Slice(missingOnLocal, func(i, j int) bool { return missingOnLocal[i].GetTime() < missingOnLocal[j].GetTime() })
	sort.Slice(missingOnRemote, func(i, j int) bool { return missingOnRemote[i].GetTime() < missingOnRemote[j].GetTime() })

	log.Printf("Index comparison: %d missing on local, %d missing on remote", len(missingOnLocal), len(missingOnRemote))

	// 3. Reconcile missing rows
	if !dryRun {
		// Copy remote -> local
		if len(missingOnLocal) > 0 {
			log.Printf("Copying %d samples from remote -> local in batches of %d...", len(missingOnLocal), batchSize)
			copied, err := copyBatch(ctx, remoteClient, localClient, missingOnLocal, batchSize)
			if err != nil {
				return nil, fmt.Errorf("copy remote -> local failed: %w", err)
			}
			stats.copiedToLocal = copied
		}

		// Copy local -> remote
		if len(missingOnRemote) > 0 {
			log.Printf("Copying %d samples from local -> remote in batches of %d...", len(missingOnRemote), batchSize)
			copied, err := copyBatch(ctx, localClient, remoteClient, missingOnRemote, batchSize)
			if err != nil {
				return nil, fmt.Errorf("copy local -> remote failed: %w", err)
			}
			stats.copiedToRemote = copied
		}
	}

	// 4. Audit remaining true gaps per source
	gapAuditStart := sinceTime
	if gapAuditStart == 0 {
		gapAuditStart = 1418915400 // Earliest sample: 2014-12-18
	}

	for _, src := range []string{"bgp1", "bgp3"} {
		gaps := 0
		// Check every 5-minute bucket up to 10 minutes ago
		for t := gapAuditStart; t < untilTime-600; t += 300 {
			key := fmt.Sprintf("%s@%d", src, t)
			_, onLocal := localMap[key]
			_, onRemote := remoteMap[key]
			if !onLocal && !onRemote {
				gaps++
			}
		}
		stats.remainingGaps[src] = gaps
	}

	return stats, nil
}

func copyBatch(ctx context.Context, srcClient, dstClient pb.BgpInfoClient, keys []*pb.SampleKey, batchSize int) (int, error) {
	totalCopied := 0
	lastLogged := 0
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		chunk := keys[i:end]

		// Fetch from source
		batchResp, err := srcClient.GetSampleBatch(ctx, &pb.SampleBatchRequest{Keys: chunk})
		if err != nil {
			return totalCopied, fmt.Errorf("GetSampleBatch failed: %w", err)
		}

		if len(batchResp.GetSamples()) == 0 {
			continue
		}

		// Insert into destination
		addRes, err := dstClient.AddSampleBatch(ctx, batchResp)
		if err != nil {
			return totalCopied, fmt.Errorf("AddSampleBatch failed: %w", err)
		}
		if !addRes.GetSuccess() {
			return totalCopied, fmt.Errorf("AddSampleBatch returned success=false")
		}

		totalCopied += len(batchResp.GetSamples())
		if totalCopied-lastLogged >= 5000 || totalCopied == len(keys) {
			log.Printf("Progress: copied %d/%d samples (%.1f%%)", totalCopied, len(keys), float64(totalCopied)/float64(len(keys))*100.0)
			lastLogged = totalCopied
		}
	}
	return totalCopied, nil
}

func printReport(s *reconcileStats, localAddr, remoteAddr string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 68))
	fmt.Println("             BGP DATABASE RECONCILIATION REPORT")
	fmt.Println(strings.Repeat("=", 68))
	fmt.Printf("Local endpoint:     %s\n", localAddr)
	fmt.Printf("Remote endpoint:    %s\n", remoteAddr)
	fmt.Printf("Window:             %s -> %s\n", s.windowStart.Format("2006-01-02 15:04:05 UTC"), s.windowEnd.Format("2006-01-02 15:04:05 UTC"))
	fmt.Printf("Local row count:    %d\n", s.localCount)
	fmt.Printf("Remote row count:   %d\n", s.remoteCount)
	fmt.Println(strings.Repeat("-", 68))
	if s.dryRun {
		fmt.Println("Mode:               DRY RUN (no rows copied)")
	} else {
		fmt.Printf("Copied Remote->Local: %d rows\n", s.copiedToLocal)
		fmt.Printf("Copied Local->Remote: %d rows\n", s.copiedToRemote)
	}
	fmt.Printf("Suspect samples:    Local: %d | Remote: %d\n", s.localSuspect, s.remoteSuspect)
	fmt.Println(strings.Repeat("-", 68))
	fmt.Println("Remaining gaps where neither DB has data:")
	for src, count := range s.remainingGaps {
		fmt.Printf("  • %-6s: %d missing 5-minute buckets\n", src, count)
	}
	fmt.Println(strings.Repeat("=", 68))
	fmt.Println("Reconciliation completed successfully.")
	fmt.Println()
}

func pingHealthcheck(url string) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("WARNING: Failed to ping Healthchecks URL (%s): %v", url, err)
		return
	}
	defer resp.Body.Close()
	log.Printf("Successfully pinged Healthchecks (status=%d)", resp.StatusCode)
}
