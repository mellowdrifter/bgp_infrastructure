package tracerpki

import (
	"fmt"
	"net/netip"
	"time"
)

type hop struct {
	hop        uint
	asNumber   int
	address    netip.Addr
	asName     string
	rDNS       string
	duration   time.Duration
	rpkiStatus string
}

func (h *hop) printHop() {
	fmt.Printf("%d  %s\t(%s)\t%d\t%s\tRPKI: %s\n", h.hop, h.rDNS, h.address, h.asNumber, h.asName, h.rpkiStatus)
}

// UNKNOWN looks confusing, so adjust output to not signed
func unknownToNotSigned(status string) string {
	if status == "UNKNOWN" {
		return "NOT SIGNED"
	}
	return status
}
