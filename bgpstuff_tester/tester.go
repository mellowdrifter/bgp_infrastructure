package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

const baseURL = "https://test.bgpstuff.net/"

var urls = []string{
	// All empty queries
	"",
	"route",
	"origin",
	"aspath",
	"roa",
	"asname",
	"sourced",
	"totals",
	"faq",

	// Queries with valid arguments
	"route?ip=8.8.8.8",
	"origin?ip=8.8.8.8",
	"aspath?ip=8.8.8.8",
	"roa?ip=8.8.8.8",
	"asname?as=15169",
	"sourced?as=15169",

	// Queries with valid requests, but invalid data (private)
	"route?ip=10.0.0.0",
	"origin?ip=10.0.0.0",
	"aspath?ip=10.0.0.0",
	"roa?ip=10.0.0.0",
	"asname?asn=4200000000",
	"sourced?asn=4200000000",

	// Queries with valid requests, but invalid data (garbage)
	"route?ip=10.0.0.0.1",
	"origin?ip=something",
	"aspath?ip=hello",
	"asname?asn=words",
	"sourced?asn=999999999999",

	// Queries with valid IPs, but no route
	"route?ip=11.0.0.0",
	"origin?ip=11.0.0.0",
	"aspath?ip=11.0.0.0",
	"roa?ip=11.0.0.0",

	// Queries with valid ASN, but no as name
	"asname?asn=4199999999",
	"sourced?asn=4199999999",
}

type result struct {
	Error    error
	Response *http.Response
	URL      string
}

func main() {

	test := flag.String("test", "", "type of test (query|load)")
	count := flag.Int("count", 1, "repeat count when load testing")

	flag.Parse()

	switch *test {
	case "query":
		runQueryTest()
	case "load":
		runLoadTest(*count)
	default:
		log.Fatal("Specify which test to run. --help if you're confused")
	}
}

func runQueryTest() {
	for _, u := range urls {
		url := fmt.Sprintf("%s%s", baseURL, u)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error getting url %s: %v\n", url, err)
			time.Sleep(2 * time.Second)
		}
		if resp.StatusCode != 200 {
			fmt.Printf("Incorrect status code for %s: %v\n", url, resp.StatusCode)
			time.Sleep(25 * time.Second)

		}
	}
	fmt.Println("All URLs checked")
}

func runLoadTest(count int) {
	for i := 0; i < count; i++ {
		var results []result
		ch := make(chan result)
		var wg sync.WaitGroup

		for _, url := range urls {
			wg.Add(1)
			go makeRequest(fmt.Sprintf("%s%s", baseURL, url), ch, &wg)
		}

		for range urls {
			results = append(results, <-ch)
		}

		wg.Wait()

		for _, v := range results {
			if v.Error != nil {
				fmt.Printf("Error received (%v) on request (%s)", v.Error, v.URL)
			}
			if v.Response.StatusCode != 200 {
				fmt.Printf("Response code (%d) on request (%s)\n", v.Response.StatusCode, v.URL)
			}
		}
		fmt.Printf("Finished loop %d\n", i+1)
	}
	fmt.Println("All done")

}

func makeRequest(url string, ch chan result, wg *sync.WaitGroup) {
	defer wg.Done()
	resp, err := http.Get(url)

	ch <- result{
		Error:    err,
		Response: resp,
		URL:      url,
	}
	return
}
