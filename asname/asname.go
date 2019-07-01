package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"golang.org/x/text/encoding/charmap"

	"github.com/golang/protobuf/proto"
	com "github.com/mellowdrifter/bgp_infrastructure/common"
	pb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpinfo"
	"google.golang.org/grpc"
)

func main() {
	defer com.TimeFunction(time.Now(), "main")

	// Locations of current ASN mapping
	urls := []string{
		"http://bgp.potaroo.net/cidr/autnums.html",
		"https://www.cidr-report.org/as2.0/autnums.html",
		//"http://10.20.30.22/asn/autnums.html",
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
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		break
	}

	if len(contents) == 0 {
		log.Fatalf("Unable to download ASN names from all URLs")
	}

	// Decode to valid UTF-8. For now these urls seem to use ISO8859_1
	decoder := charmap.ISO8859_1.NewDecoder()
	output, _ := decoder.Bytes(contents)
	strcontents := string(output)

	// I only want the AS number and name assigned
	reg := regexp.MustCompile(`AS(\d+)\s*</a> (.*),`)

	// -1 means no limit of matches
	res := reg.FindAllStringSubmatch(strcontents, -1)

	// pack into proto so we can send off to bgpinfo
	largest := 0
	var bigName []byte
	var asnNames []*pb.AsnName
	for _, AS := range res {
		if len(AS[2]) > largest {
			largest = len(AS[2])
			bigName = []byte(AS[2])
		}
		asnNames = append(asnNames, &pb.AsnName{
			AsNumber: com.StringToUint32(AS[1]),
			AsName:   AS[2],
		})
	}

	fmt.Printf("Longest name is %s and it's length is %d\n", bigName, largest)
	//fmt.Printf("There are a total of %d AS numbers\n", len(asnNames))
	log.Println("Finished downloading and packing")

	// gRPC dial and send data
	// TODO: This should go into config.ini
	conn, err := grpc.Dial("127.0.0.1:7179", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Unable to dial gRPC server: %s", err)
	}
	defer conn.Close()
	c := pb.NewBgpInfoClient(conn)

	resp, err := c.UpdateAsnames(context.Background(), &pb.AsnamesRequest{
		AsnNames: asnNames,
	})
	if err != nil {
		log.Fatalf("Unable to send proto: %s", err)
	}

	fmt.Println(proto.MarshalTextString(resp))
}
