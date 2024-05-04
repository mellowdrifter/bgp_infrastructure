package main

import (
	"os"
	"reflect"
	"testing"

	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
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
		AsName:   "ALTECOM Alta Tecnologia en Comunicacions, S.L",
		AsNumber: 16030,
		AsLocale: "ES",
	},
	{
		AsName:   "WARNETCZ-AS Warnet.cz s.r.o.",
		AsNumber: 47727,
		AsLocale: "CZ",
	},
	{
		AsName:   "COOPERATIVA TELEFONICA Y OTROS SERVICIOS PUBLICOS  ASISTENCIALES, EDUCATIVOS, VIVIENDA, CREDITO Y CONSUMO TILISARAO LIMITADA",
		AsNumber: 267925,
		AsLocale: "AR",
	},
	{
		AsName:   "VRSN-AC50-340 - VeriSign Global Registry Services",
		AsNumber: 396632,
		AsLocale: "US",
	},
	{
		AsName:   "GSCS",
		AsNumber: 397335,
		AsLocale: "CA",
	},
	{
		AsName:   "SAAQ-PROD - Societe de l'Assurance Automobile du Quebec",
		AsNumber: 397421,
		AsLocale: "CA",
	},
	{
		AsName:   "-Reserved AS-",
		AsNumber: 397759,
		AsLocale: "ZZ",
	},
}

func TestDecoder(t *testing.T) {
	data, err := os.ReadFile("autnums.html")
	if err != nil {
		panic(err)
	}
	output := decoder(data)
	for i := 0; i < len(output); i++ {
		if !reflect.DeepEqual(output[i], good[i]) {
			t.Errorf("No match error. Wanted %v, got %v", good[i], output[i])
			continue
		}
	}
}

func TestDecoderFull(t *testing.T) {
	data, err := os.ReadFile("autnums.html")
	if err != nil {
		panic(err)
	}
	output := decoder(data)
	if len(output) != count {
		t.Errorf("Amount of ASs should be %d, but got %d", count, len(output))
	}
	for _, info := range output {
		if info.GetAsLocale() == "" {
			t.Errorf("AS %s has no Locale", info.GetAsName())
		}
	}
}

func TestGetTextASNs(t *testing.T) {
	data, err := os.ReadFile("asn.txt")
	if err != nil {
		panic(err)
	}
	output := decodeText(data)
	for i := 0; i < len(output); i++ {
		if !reflect.DeepEqual(output[i], good[i]) {
			t.Errorf("No match error. Wanted %v, got %v", good[i], output[i])
			continue
		}
	}
}
