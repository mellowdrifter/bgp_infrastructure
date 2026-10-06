package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"strings"

	com "github.com/mellowdrifter/bgp_infrastructure/pkg/common"
)

type validationConfig struct {
	v4Min          uint32
	v4Max          uint32
	v6Min          uint32
	v6Max          uint32
	maxDeltaPct    float64
	maxPeerDiffPct float64
	minPeerRatio   float64
}

func defaultValidationConfig() validationConfig {
	return validationConfig{
		v4Min:          800000,
		v4Max:          1600000,
		v6Min:          150000,
		v6Max:          600000,
		maxDeltaPct:    3.0,
		maxPeerDiffPct: 5.0,
		minPeerRatio:   0.5,
	}
}

// validateSample validates an incoming sample against configured bounds,
// peer session health, and historical / cross-source consistency.
func validateSample(ctx context.Context, db *sql.DB, b *com.BgpUpdate, cfg validationConfig) (string, string) {
	if db == nil {
		return "ok", ""
	}

	var reasons []string

	// 1. Check absolute bounds for IPv4
	if b.V4Count < cfg.v4Min || b.V4Count > cfg.v4Max {
		reasons = append(reasons, fmt.Sprintf("v4 count %d outside bounds [%d, %d]", b.V4Count, cfg.v4Min, cfg.v4Max))
	}

	// 2. Check absolute bounds for IPv6
	if b.V6Count < cfg.v6Min || b.V6Count > cfg.v6Max {
		reasons = append(reasons, fmt.Sprintf("v6 count %d outside bounds [%d, %d]", b.V6Count, cfg.v6Min, cfg.v6Max))
	}

	// 3. Check peer session health
	if b.PeersConfigured > 0 {
		if b.PeersUp == 0 {
			reasons = append(reasons, fmt.Sprintf("0/%d v4 peers up", b.PeersConfigured))
		} else if float64(b.PeersUp)/float64(b.PeersConfigured) < cfg.minPeerRatio {
			reasons = append(reasons, fmt.Sprintf("v4 peers degraded: %d/%d up (%.1f%%)",
				b.PeersUp, b.PeersConfigured, float64(b.PeersUp)/float64(b.PeersConfigured)*100))
		}
	}

	// 4. Check step change vs previous sample from the same source
	var prevV4, prevV6 uint32
	err := db.QueryRowContext(ctx,
		`SELECT V4COUNT, V6COUNT FROM INFO WHERE source = ? AND TIME < ? ORDER BY TIME DESC LIMIT 1`,
		b.Source, b.Time,
	).Scan(&prevV4, &prevV6)

	if err == nil && prevV4 > 0 {
		v4DeltaPct := math.Abs(float64(int64(b.V4Count)-int64(prevV4))) / float64(prevV4) * 100
		if v4DeltaPct > cfg.maxDeltaPct {
			// Check cross-collector consensus: did another source record at the exact same timestamp?
			var otherSource string
			var otherV4 uint32
			errCross := db.QueryRowContext(ctx,
				`SELECT source, V4COUNT FROM INFO WHERE source != ? AND TIME = ? LIMIT 1`,
				b.Source, b.Time,
			).Scan(&otherSource, &otherV4)

			if errCross == nil && otherV4 > 0 {
				crossDiffPct := math.Abs(float64(int64(b.V4Count)-int64(otherV4))) / float64(otherV4) * 100
				if crossDiffPct > cfg.maxPeerDiffPct {
					reasons = append(reasons, fmt.Sprintf("v4 shifted %.1f%% vs prev (%d->%d); diverged %.1f%% from %s (%d)",
						v4DeltaPct, prevV4, b.V4Count, crossDiffPct, otherSource, otherV4))
				} else {
					log.Printf("Cross-source consensus at TIME=%d: both %s and %s saw v4 shift (diff=%.1f%% <= %.1f%%)",
						b.Time, b.Source, otherSource, crossDiffPct, cfg.maxPeerDiffPct)
				}
			} else {
				reasons = append(reasons, fmt.Sprintf("v4 shifted %.1f%% vs prev (%d->%d)",
					v4DeltaPct, prevV4, b.V4Count))
			}
		}

		if prevV6 > 0 {
			v6DeltaPct := math.Abs(float64(int64(b.V6Count)-int64(prevV6))) / float64(prevV6) * 100
			if v6DeltaPct > cfg.maxDeltaPct {
				reasons = append(reasons, fmt.Sprintf("v6 shifted %.1f%% vs prev (%d->%d)",
					v6DeltaPct, prevV6, b.V6Count))
			}
		}
	}

	if len(reasons) > 0 {
		return "suspect", strings.Join(reasons, "; ")
	}

	return "ok", ""
}
