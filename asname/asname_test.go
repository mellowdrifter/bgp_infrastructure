package main

import (
	"io/ioutil"
	"reflect"
	"testing"

	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpinfo"
)

const count = 11

var good = []*pb.AsnName{
	&pb.AsnName{
		AsName:   "-Reserved AS-",
		AsLocale: "ZZ",
	},
	&pb.AsnName{
		AsName:   "US-NATIONAL-INSTITUTE-OF-STANDARDS-AND-TECHNOLOGY",
		AsNumber: 49,
		AsLocale: "US",
	},
	&pb.AsnName{
		AsName:   "DNIC-ASBLK-05120-05376 - DoD Network Information Center",
		AsNumber: 5218,
		AsLocale: "US",
	},
	&pb.AsnName{
		AsName:   "ARRIS-TECHNOLOGY-SD-NOC - ARRIS Technology, Inc.",
		AsNumber: 10580,
		AsLocale: "US",
	},
	&pb.AsnName{
		AsName:   "ALTECOM",
		AsNumber: 16030,
		AsLocale: "ES",
	},
	&pb.AsnName{
		AsName:   "WARNETCZ-AS Warnet.cz s.r.o.",
		AsNumber: 47727,
		AsLocale: "CZ",
	},
	&pb.AsnName{
		AsName:   "COOPERATIVA TELEFONICA Y OTROS SERVICIOS PUBLICOS  ASISTENCIALES, EDUCATIVOS, VIVIENDA, CREDITO Y CONSUMO TILISARAO LIMITADA",
		AsNumber: 267925,
		AsLocale: "AR",
	},
	&pb.AsnName{
		AsName:   "VRSN-AC50-340 - VeriSign Global Registry Services",
		AsNumber: 396632,
		AsLocale: "US",
	},
	&pb.AsnName{
		AsName:   "GSCS - St. Paul's Roman Catholic Separate School Division #20",
		AsNumber: 397335,
		AsLocale: "CA",
	},
	&pb.AsnName{
		AsName:   "SAAQ-PROD - Societe de l'Assurance Automobile du Quebec",
		AsNumber: 397421,
		AsLocale: "CA",
	},
	&pb.AsnName{
		AsName:   "-Reserved AS-",
		AsNumber: 397723,
		AsLocale: "ZZ",
	},
}

func TestDecoder(t *testing.T) {
	data, err := ioutil.ReadFile("autnums.html")
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
	data, err := ioutil.ReadFile("autnums.html")
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
