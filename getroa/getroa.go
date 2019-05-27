package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	rpki    = "https://rpki.cloudflare.com/rpki.json"
	logfile = "/var/log/rpki_update.log"
	v4File  = "/etc/bird/v4.roa"
	v6File  = "/etc/bird/v6.roa"
)

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
	// set up log file
	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open logfile: %v\n", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// get and divide up the ROAs
	log.Printf("Downloading %s\n", path.Base(rpki))
	fullroas := getROAs()
	v4, v6 := divideV4V6(fullroas)

	log.Printf("There are %d IPv4 roas and %d IPv6 roas\n", len(*v4), len(*v6))

	// write updated roas to file and refresh bird
	err = writeROAs(v4, v4File)
	if err != nil {
		log.Printf("unable to write IPv4 ROAs: %v", err)
	}
	log.Printf("file written")
	cmd := exec.Command("birdc", "'configure'")
	err = cmd.Run()
	if err != nil {
		log.Printf("unable to reconfigure bird: %v", err)
	}
	log.Printf("bird reconfigured")

	err = writeROAs(v6, v6File)
	if err != nil {
		log.Printf("unable to write IPv6 ROAs: %v", err)
	}
	cmd = exec.Command("birdc6", "'configure'")
	err = cmd.Run()
	if err != nil {
		log.Printf("unable to reconfigure bird6: %v", err)
	}
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

func divideV4V6(r *rpkiResponse) (*[]roa, *[]roa) {
	var v4roa []roa
	var v6roa []roa

	for _, roa := range r.Roas {
		if strings.Contains(roa.Prefix, ":") {
			v6roa = append(v6roa, roa)
		} else {
			v4roa = append(v4roa, roa)
		}
	}

	return &v4roa, &v6roa
}

func writeROAs(roas *[]roa, filename string) error {
	tmpfile := "/tmp/roa"
	file, err := os.Create(tmpfile)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, roa := range *roas {
		update := fmt.Sprintf("roa %s max %d as %s;\n", roa.Prefix, int(roa.Mask), roa.Asn[2:])
		fmt.Fprintf(file, update)
	}
	os.Rename(tmpfile, filename)

	return nil
}
