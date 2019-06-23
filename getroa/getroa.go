package main

import (
	"bytes"
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
	if err = writeROAs(v4, v4File); err != nil {
		log.Printf("unable to write IPv4 ROAs: %v", err)
	}
	if err = reloadBird("birdc"); err != nil {
		log.Printf("Unable to refresh bird: %v", err)
	}

	if err = writeROAs(v6, v6File); err != nil {
		log.Printf("unable to write IPv6 ROAs: %v", err)
	}
	if err = reloadBird("birdc6"); err != nil {
		log.Printf("Unable to refresh bird: %v", err)
	}
}

func reloadBird(d string) error {
	cmd := fmt.Sprintf("/usr/sbin/%s", d)
	cmdArg := []string{"configure"}
	cmdOut, err := exec.Command(cmd, cmdArg...).Output()
	if err != nil {
		return err
	}
	if !bytes.Contains(cmdOut, []byte("Reconfigured")) {
		return fmt.Errorf("%s not refreshed", d)
	}
	log.Printf("%s reconfigured", d)
	return nil
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
	// TODO: tmpfile should maybe be in-memory? Why did I do it this way?
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
	if err = os.Rename(tmpfile, filename); err != nil {
		return err
	}

	return nil
}
