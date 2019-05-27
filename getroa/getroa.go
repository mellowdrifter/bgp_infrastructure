package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
)

const rpki = "https://rpki.cloudflare.com/rpki.json"

var example = []byte(`
{
    "metadata": {
        "counts": 87047,
        "generated": 1558887717,
        "valid": 1558891317,
        "signature": "3045022100d5a238308e8bdd6798e91136295c601ce2a02d09188362cfcec20a621b0ee2380220051398d6058c5b7b8253c320fd9981a221daf9cbf9f025e4f7043364dff8d24a",
        "signatureDate": "3045022100c3ffdc1e4320dec5c88e64fd12cbace6aa1e1147b800ab30e607a37ed90152200220177ffe1a466afc3bb8ea72f5a2d3b80f85b997f773d5bca8f9c6d92d41d15b93"
    },
    "roas": [
        {
            "prefix": "154.16.59.0/24",
            "maxLength": 24,
            "asn": "AS132906",
            "ta": "Cloudflare - AFRINIC"
        },
        {
            "prefix": "154.127.54.0/23",
            "maxLength": 24,
            "asn": "AS13768",
            "ta": "Cloudflare - AFRINIC"
        },
        {
            "prefix": "2001:43f8:110::/48",
            "maxLength": 48,
            "asn": "AS37181",
            "ta": "Cloudflare - AFRINIC"
		}
	]
}
`)

type rpkiResponse struct {
	metadata `json:"metadata"`
	roas
}

type metadata struct {
	Generated float64 `json:"generated"`
	Valid     float64 `json:"valid"`
}

type roas struct {
	Roas []roa `json:"roas"`
}

type roa struct {
	Prefix string  `json:"prefix"`
	Mask   float64 `json:"maxLength"`
	Asn    string  `json:"asn"`
}

func main() {
	fmt.Printf("Downloading %s\n", path.Base(rpki))
	fullroas := getROAs()

	v4, v6 := divideV4V6(fullroas)

	fmt.Printf("There are %d IPv4 roas and %d IPv6 roas\n", len(v4), len(v6))
}

func getROAs() *rpkiResponse {
	resp, err := http.Get(rpki)
	if err != nil {
		os.Exit(1)
	}

	defer resp.Body.Close()
	roaJSON, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		os.Exit(1)
	}

	r := new(rpkiResponse)
	json.Unmarshal(roaJSON, &r)

	return r

}

func divideV4V6(r *rpkiResponse) ([]roa, []roa) {
	var v4roa []roa
	var v6roa []roa

	for _, roa := range r.Roas {
		if strings.Contains(roa.Prefix, ":") {
			v6roa = append(v6roa, roa)
		} else {
			v4roa = append(v4roa, roa)
		}
	}

	return v4roa, v6roa

}
