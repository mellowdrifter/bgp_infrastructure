package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"

	"golang.org/x/text/encoding/charmap"
	"gopkg.in/ini.v1"

	"github.com/golang/protobuf/proto"
	com "github.com/mellowdrifter/bgp_infrastructure/common"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
	"google.golang.org/grpc"
)

func main() {
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
	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
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

	// Send update
	resp, err := c.UpdateAsnames(context.Background(), req)
	if err != nil {
		log.Fatalf("Unable to send proto: %s", err)
	}

	log.Printf("Updated database with response %s", proto.MarshalTextString(resp))
}

func getASNs() (*pb.AsnamesRequest, error) {
	// Locations of current ASN mapping
	textUrl := "https://ftp.ripe.net/ripe/asnames/asn.txt"
	urls := []string{
		"http://bgp.potaroo.net/cidr/autnums.html",
		"https://www.cidr-report.org/as2.0/autnums.html",
	}

	res, err := getASNFromUrl(textUrl, true)
	if err == nil {
		log.Println("returning asnames")
		return &pb.AsnamesRequest{
			AsnNames: res,
		}, nil
	}
	log.Printf("unable to decode text url, moving on: %v\n", err)

	for _, url := range urls {
		// There are two URLs to check. If error on the first, try the second.
		res, err := getASNFromUrl(url, false)
		if err != nil {
			log.Printf("got error on url(%s): %s\n", url, err)
			continue
		}
		log.Println("returning asnames")
		return &pb.AsnamesRequest{
			AsnNames: res,
		}, nil
	}

	return nil, errors.New("unable to download any ASNs")
}

func getASNFromUrl(url string, isText bool) ([]*pb.AsnName, error) {
	var contents []byte
	log.Printf("Downloading AS list from %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("got status code error on url(%s): %d\n", url, resp.StatusCode)
	}
	defer resp.Body.Close()
	contents, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if isText {
		return decodeText(contents), nil
	}
	return decodeHTML(contents), nil
}

func decodeText(data []byte) []*pb.AsnName {
	reg := regexp.MustCompile(`(\d+)\s(.*),\s*([A-Z]{2})`)
	var asnNames []*pb.AsnName

	res := reg.FindAllStringSubmatch(string(data), -1)
	for _, as := range res {
		asnNames = append(asnNames, &pb.AsnName{
			AsNumber: com.StringToUint32(as[1]),
			AsName:   as[2],
			AsLocale: as[3],
		})
	}

	return asnNames
}

func decodeHTML(contents []byte) []*pb.AsnName {
	reg := regexp.MustCompile(`AS(\d+)\s*</a> (.*),\s*([A-Z]{2})`)
	var asnNames []*pb.AsnName

	// Decode to valid UTF-8. For now these urls seem to use ISO8859_1
	decoder := charmap.ISO8859_1.NewDecoder()
	output, _ := decoder.Bytes(contents)

	res := reg.FindAllStringSubmatch(string(output), -1)
	for _, as := range res {
		asnNames = append(asnNames, &pb.AsnName{
			AsNumber: com.StringToUint32(as[1]),
			AsName:   as[2],
			AsLocale: as[3],
		})
	}

	return asnNames
}
