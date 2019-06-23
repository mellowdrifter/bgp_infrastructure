package main

import (
	"encoding/json"
	"testing"
)

var rpkijsonResponse = []byte(`
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

func TestDivideV4V6(t *testing.T) {
	data := new(rpkiResponse)
	json.Unmarshal(rpkijsonResponse, &data)
	v4, v6 := divideV4V6(data)

	if len(*v4) != 2 {
		t.Fatalf("supposed to get 2 IPv4 addresses, but got %d", len(*v4))
	}
	if len(*v6) != 1 {
		t.Fatalf("supposed to get 1 IPv6 address, but got %d", len(*v6))
	}
}
