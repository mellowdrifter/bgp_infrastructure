package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"time"

	"golang.org/x/text/encoding/charmap"
	"gopkg.in/ini.v1"

	"github.com/golang/protobuf/proto"
	com "github.com/mellowdrifter/bgp_infrastructure/common"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpinfo"
	"google.golang.org/grpc"
)

func main() {
	defer com.TimeFunction(time.Now(), "main")

	// load in config
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	path := fmt.Sprintf("%s/config.ini", path.Dir(exe))
	cf, err := ini.Load(path)
	if err != nil {
		log.Fatalf("failed to read config file: %v\n", err)
	}

	logfile := cf.Section("log").Key("logfile").String()
	bgpinfo := cf.Section("bgpinfo").Key("server").String()

	// Set up log file
	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open logfile: %v\n", err)
	}
	defer f.Close()
	log.SetOutput(f)

	req, err := getASNs()
	if err != nil {
		log.Fatalf("Error received: %s", err)
	}

	// gRPC dial and send data
	conn, err := grpc.Dial(bgpinfo, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Unable to dial gRPC server: %s", err)
	}
	defer conn.Close()
	c := pb.NewBgpInfoClient(conn)

	resp, err := c.UpdateAsnames(context.Background(), req)
	if err != nil {
		log.Fatalf("Unable to send proto: %s", err)
	}

	log.Println(proto.MarshalTextString(resp))
}

func getASNs() (*pb.AsnamesRequest, error) {
	// Locations of current ASN mapping
	urls := []string{
		"http://bgp.potaroo.net/cidr/autnums.html",
		"https://www.cidr-report.org/as2.0/autnums.html",
	}

	// Download list and print error if unable to get
	var contents []byte
	log.Println("Downloading AS list")
	for _, url := range urls {
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Got error on url(%s): %s\n", url, err)
			continue
		}
		if resp.StatusCode != 200 {
			log.Printf("Got status code error on url(%s): %d\n", url, resp.StatusCode)
			continue
		}
		defer resp.Body.Close()
		contents, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return &pb.AsnamesRequest{}, fmt.Errorf("Error reading URL: %v", err)
		}
		break
	}

	if len(contents) == 0 {
		return &pb.AsnamesRequest{}, errors.New("Unable to download ASN names from all URLs")
	}

	asnNames := decoder(contents)

	log.Println("Finished downloading and packing")

	return &pb.AsnamesRequest{
		AsnNames: asnNames,
	}, nil

}

func decoder(contents []byte) []*pb.AsnName {
	// Decode to valid UTF-8. For now these urls seem to use ISO8859_1
	decoder := charmap.ISO8859_1.NewDecoder()
	output, _ := decoder.Bytes(contents)
	strcontents := string(output)

	// I need the AS name, AS number, and the Locale.
	reg := regexp.MustCompile(`AS(\d+)\s*</a> (.*),\s*([A-Z]{2})`)

	// -1 means no limit of matches
	res := reg.FindAllStringSubmatch(strcontents, -1)

	// pack into proto so we can send off to bgpinfo
	var asnNames []*pb.AsnName
	for _, AS := range res {
		asnNames = append(asnNames, &pb.AsnName{
			AsNumber: com.StringToUint32(AS[1]),
			AsName:   AS[2],
			AsLocale: AS[3],
		})
	}

	return asnNames

}
