package main

import (
	"os"
	"testing"

	pb "github.com/mellowdrifter/bgp_infrastructure/internal/bgpsql"
)

const count = 11

var good = []*pb.AsnName{
	{
		AsName:   "-Reserved AS-",
		AsLocale: "ZZ",
	},
	{
		AsName:   "US-NATIONAL-INSTITUTE-OF-STANDARDS-AND-TECHNOLOGY",
		AsNumber: 49,
		AsLocale: "US",
	},
	{
		AsName:   "DNIC-ASBLK-05120-05376",
		AsNumber: 5218,
		AsLocale: "US",
	},
	{
		AsName:   "ARRIS-TECHNOLOGY-SD-NOC",
		AsNumber: 10580,
		AsLocale: "US",
	},
	{
		AsName:   "ALTECOM",
		AsNumber: 16030,
		AsLocale: "ES",
	},
	{
		AsName:   "WARNETCZ-AS Warnet.cz s.r.o.",
		AsNumber: 47727,
		AsLocale: "CZ",
	},
	{
		AsName:   "COOPERATIVA TELEFONICA Y OTROS SERVICIOS PUBLICOS ASISTENCIALES, EDUCATIVOS, VIVIENDA, CREDITO Y CONSUMO TILISARAO LIMITADA",
		AsNumber: 267925,
		AsLocale: "AR",
	},
	{
		AsName:   "VRSN-AC50-340",
		AsNumber: 396632,
		AsLocale: "US",
	},
	{
		AsName:   "GSCS",
		AsNumber: 397335,
		AsLocale: "CA",
	},
	{
		AsName:   "SAAQ-PROD",
		AsNumber: 397421,
		AsLocale: "CA",
	},
	{
		AsName:   "LYNK-AS",
		AsNumber: 397759,
		AsLocale: "US",
	},
	{
		AsName:   "IQVIA-DURHAM",
		AsNumber: 402332,
		AsLocale: "US",
	},
	{
		AsName:   "@SONET COLOMBIA SAS",
		AsNumber: 274054,
		AsLocale: "CO",
	},
	{
		AsName:   "PEACEWEB-GROUP-AS This AS number is used by organizations from PeaceWeb Group. For any abuse-related issues, please send us a message at abuse@peaceweb.com.",
		AsNumber: 210907,
		AsLocale: "NL",
	},
}

func TestDecoder(t *testing.T) {
	hData, err := os.ReadFile("autnums.html")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	hOutput := decodeHTML(hData)

	tData, err := os.ReadFile("asn.txt")
	if err != nil {
		panic(err)
	}
	tOutput := decodeText(tData)

	// Build maps for quick lookups
	hAsnMap := make(map[uint32]*pb.AsnName, len(hOutput))
	for _, asn := range hOutput {
		hAsnMap[asn.GetAsNumber()] = asn
	}
	tAsnMap := make(map[uint32]*pb.AsnName, len(tOutput))
	for _, asn := range tOutput {
		tAsnMap[asn.GetAsNumber()] = asn
	}
	if len(hOutput) != len(tOutput) {
		t.Errorf("Mismatch in number of ASNs: %d != %d", len(hOutput), len(tOutput))
	}

	for _, asn := range good {
		hRef := hAsnMap[asn.GetAsNumber()]
		tRef := tAsnMap[asn.GetAsNumber()]
		if hRef == nil {
			t.Errorf("AS %d not found in map", asn.GetAsNumber())
			continue
		}
		if tRef == nil {
			t.Errorf("AS %d not found in map", asn.GetAsNumber())
			continue
		}
		if hRef.GetAsName() != asn.GetAsName() {
			t.Errorf("AS %d name mismatch: %q != %q", asn.GetAsNumber(), hRef.GetAsName(), asn.GetAsName())
		}
		if hRef.GetAsLocale() != asn.GetAsLocale() {
			t.Errorf("AS %d locale mismatch: %q != %q", asn.GetAsNumber(), hRef.GetAsLocale(), asn.GetAsLocale())
		}
		if tRef.GetAsName() != asn.GetAsName() {
			t.Errorf("AS %d name mismatch: %q != %q", asn.GetAsNumber(), tRef.GetAsName(), asn.GetAsName())
		}
		if tRef.GetAsLocale() != asn.GetAsLocale() {
			t.Errorf("AS %d locale mismatch: %q != %q", asn.GetAsNumber(), tRef.GetAsLocale(), asn.GetAsLocale())
		}
	}
}
