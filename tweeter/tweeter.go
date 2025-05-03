package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"math/rand"

	"github.com/ChimeraCoder/anaconda"
	"github.com/mellowdrifter/bgp_infrastructure/bsky"
	bpb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
	gpb "github.com/mellowdrifter/bgp_infrastructure/proto/grapher"
	"github.com/mellowdrifter/gotwi"
	"github.com/mellowdrifter/gotwi/tweet/managetweet"
	"github.com/mellowdrifter/gotwi/tweet/managetweet/types"

	"github.com/mattn/go-mastodon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/ini.v1"
)

const (
	// If I see IPv4 and IPv6 values less than these values, there is an issue.
	// This value can be revised once every 6 months or so.
	minV4 = 950000
	minV6 = 200000
)

type tweet struct {
	account string
	message string
	media   []byte
	video   []byte
}

type toTweet struct {
	// tableSize tweets the size and delta every 6 hours.
	tableSize bool

	// graph will plot the changes over various time ranges.
	weekGraph     bool
	monthGraph    bool
	sixMonthGraph bool
	annualGraph   bool

	subnetPie bool

	rpkiPie bool

	test bool
}

type config struct {
	grapher string
	servers []string
	file    *ini.File
	dryRun  bool
}

type tweeter struct {
	mux *http.ServeMux
	mu  sync.Mutex
	cfg config
}

func Setup() (config, error) {
	// load in config
	exe, err := os.Executable()
	if err != nil {
		return config{}, err
	}
	path := fmt.Sprintf("%s/config.ini", path.Dir(exe))
	cf, err := ini.ShadowLoad(path)
	if err != nil {
		log.Fatalf("failed to read config file: %v\n", err)
	}

	var config config

	config.file = cf

	gr := cf.Section("grapher").Key("server").String()
	config.grapher = fmt.Sprintf("%s:443", gr)
	config.servers = cf.Section("bgpinfo").Key("server").ValueWithShadows()

	flag.Parse()

	return config, nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg, err := Setup()
	if err != nil {
		log.Fatalf("unable to set things up: %v", err)
	}

	var srv tweeter
	srv.mux = http.NewServeMux()
	srv.cfg = cfg

	srv.mux.HandleFunc("/post", srv.post())
	srv.mux.HandleFunc("/testbksy", srv.testbsky())
	srv.mux.HandleFunc("/", srv.dryrun())
	srv.mux.HandleFunc("/favicon.ico", faviconHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("*** Service Started on Port %s ***\n", port)
	log.Fatal(http.ListenAndServe(":"+port, srv.mux))
}

// Required to implement the interface.
func (t *tweeter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.mux.ServeHTTP(w, r)
}

// ignore the request to favicon when I'm calling through a browser.
func faviconHandler(w http.ResponseWriter, r *http.Request) {}

// Test bsky
func (t *tweeter) testbsky() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.mu.Lock()
		defer t.mu.Unlock()
		t.cfg.dryRun = true

		var todo toTweet
		todo.tableSize = true

		tweetList, err := getTweets(todo, t.cfg)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "unable to get tweets: %v", err)
			return
		}
		for _, tweet := range tweetList {
			// Post it
			if err := postBsky(tweet, t.cfg.file); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Printf("error when posting to bluesky: %v", err)
			}
		}

	}
}

// Basic index for now.
func (t *tweeter) dryrun() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("This is the full request: %#v\n", r)
		log.Printf("url is %v\n", r.RequestURI)
		t.mu.Lock()
		defer t.mu.Unlock()
		t.cfg.dryRun = true

		var todo toTweet
		todo.tableSize = true
		todo.rpkiPie = true
		todo.subnetPie = true
		todo.weekGraph = true
		todo.monthGraph = true
		todo.sixMonthGraph = true
		todo.annualGraph = true
		log.Printf("todo: %#v\n", todo)

		tweetList, err := getTweets(todo, t.cfg)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "unable to get tweets: %v", err)
			return
		}

		fmt.Fprintf(w, "<h1>What will I tweet?</h1>")
		fmt.Fprintf(w, "<p>If I were to run now, this is what I would tweet: %#v</p>\n", todo)
		fmt.Fprintf(w, "<p>The time is now %v</p>\n", time.Now())

		for i, tweet := range tweetList {
			fmt.Fprintf(w, "<p><b>tweet %d: %s</b></p>\n", i, tweet.message)
			if len(tweet.media) > 0 {
				image64 := base64.StdEncoding.EncodeToString(tweet.media)
				fmt.Fprintf(w, fmt.Sprintf(`<img src="data:image/png;base64,%s">`, image64))
			}
		}
	}
}

// post is the function that will really post things!
func (t *tweeter) post() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("This is the full request: %#v\n", r)
		log.Printf("url is %v\n", r.RequestURI)
		t.mu.Lock()
		defer t.mu.Unlock()

		todo := whatToTweet(time.Now())

		t.cfg.dryRun = false

		tweetList, err := getTweets(todo, t.cfg)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "unable to get tweets: %v", err)
			return
		}

		if len(tweetList) == 0 {
			fmt.Fprintf(w, "nothing to tweet at this time")
			return
		}

		for _, tweet := range tweetList {
			// bsky is not showing the second post often. Maybe I should sleep between posts?
			time.Sleep(10 * time.Second)
			// Tweet it
			if err := postTweet(tweet, t.cfg.file); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Printf("error when tweeting: %v", err)
			}
			// Post it
			if err := postBsky(tweet, t.cfg.file); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Printf("error when posting to bluesky: %v", err)
			}
		}
	}
}

// getTweets will compile all tweets as according to the todo list of tweets.
func getTweets(todo toTweet, cfg config) ([]tweet, error) {
	var listOfTweets []tweet
	/*
		if todo.test {
			tweets := []tweet{
				{
					account: "bgp4table",
					message: "I'm alive",
				},
				{
					account: "bgp6table",
					message: "I'm alive",
				},
			}
			listOfTweets = append(listOfTweets, tweets...)
		}
	*/

	if todo.tableSize {
		tweets, err := allCurrent(cfg)
		if err != nil {
			return listOfTweets, fmt.Errorf("unable to gather table size info: %v", err)
		}
		listOfTweets = append(listOfTweets, tweets...)
	}
	if todo.weekGraph {
		tweets, err := movement(cfg, bpb.MovementRequest_WEEK)
		if err != nil {
			return listOfTweets, fmt.Errorf("unable to gather weekly graph info: %v", err)
		}
		listOfTweets = append(listOfTweets, tweets...)
	}
	if todo.monthGraph {
		tweets, err := movement(cfg, bpb.MovementRequest_MONTH)
		if err != nil {
			return listOfTweets, fmt.Errorf("unable to gather monthly graph info: %v", err)
		}
		listOfTweets = append(listOfTweets, tweets...)
	}
	if todo.sixMonthGraph {
		tweets, err := movement(cfg, bpb.MovementRequest_SIXMONTH)
		if err != nil {
			return listOfTweets, fmt.Errorf("unable to gather six monthly graph info: %v", err)
		}
		listOfTweets = append(listOfTweets, tweets...)
	}
	if todo.annualGraph {
		tweets, err := movement(cfg, bpb.MovementRequest_ANNUAL)
		if err != nil {
			return listOfTweets, fmt.Errorf("unable to gather six monthly graph info: %v", err)
		}
		listOfTweets = append(listOfTweets, tweets...)
	}
	if todo.rpkiPie {
		tweets, err := rpki(cfg)
		if err != nil {
			return listOfTweets, fmt.Errorf("unable to generate RPKI tweets: %v", err)
		}
		listOfTweets = append(listOfTweets, tweets...)
	}
	if todo.subnetPie {
		tweets, err := subnets(cfg)
		if err != nil {
			return listOfTweets, fmt.Errorf("unable to generate subnet pie tweets: %v", err)
		}
		listOfTweets = append(listOfTweets, tweets...)
	}

	return listOfTweets, nil
}

// whatToTweet will determine exactly what information should be tweeted. This
// is all determined by the time and day on which it's called.
func whatToTweet(now time.Time) toTweet {
	var todo toTweet

	// Only tweet items if called in valid hours.
	validHours := map[int]bool{
		2:  true,
		8:  true,
		14: true,
		20: true,
	}
	if !validHours[now.Hour()] {
		todo.test = true
		return todo
	}

	// Table size is tweeted every 6 hours, every day.
	todo.tableSize = true

	// I only set the rest at 20:00 UTC, any other time we should return immidiately.
	if now.Hour() != 20 {
		return todo
	}

	// Weekly growth graph every Monday.
	todo.weekGraph = (now.Weekday() == time.Monday)

	// Monthly graphs on the first day of the month.
	// Unless the first falls on a Weekend
	if now.Day() == 1 && isNotWeekend(now.Weekday()) {
		todo.monthGraph = true
	}
	if (now.Day() == 2 || now.Day() == 3) && now.Weekday() == time.Monday {
		todo.monthGraph = true
	}

	// 1st of July also tweet a 6 month growth graph.
	if now.Day() == 1 && now.Month() == time.July && isNotWeekend(now.Weekday()) {
		todo.sixMonthGraph = true
	}
	if (now.Day() == 2 || now.Day() == 3) && now.Month() == time.July && now.Weekday() == time.Monday {
		todo.sixMonthGraph = true
	}

	// Annual graph. Post on 6th of January as no-one is around in the beginning.
	if now.Month() == time.January && now.Day() == 6 && isNotWeekend(now.Weekday()) {
		todo.annualGraph = true
	}
	if now.Month() == time.January && (now.Day() == 7 || now.Day() == 8) && now.Weekday() == time.Monday {
		todo.annualGraph = true
	}

	// On Wednesday I tweet the subnet pie graph.
	todo.subnetPie = (now.Weekday() == time.Wednesday)

	// On Thursday I tweet the RPKI status.
	todo.rpkiPie = (now.Weekday() == time.Thursday)

	return todo
}

func isNotWeekend(t time.Weekday) bool {
	if t == time.Saturday || t == time.Sunday {
		return false
	}
	return true
}

// getConnection will return a connection to a gRPC server. Caller should close.
func getConnection(srv string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(srv, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("unable to dial gRPC server: %v", err)
	}
	return conn, err
}

// getTLSConnection is the same as getConnection, but it uses TLS as an option
// as is required by Google Cloud Run.
func getTLSConnection(srv string) (*grpc.ClientConn, error) {
	creds := credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
	})
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}
	tconn, err := grpc.Dial(srv, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to dial gRPC server: %v", err)
	}

	return tconn, nil
}

// getLiveServer will return the first live connection it can get. If neither server
// can be dialed, an error is returned.
// TODO: Just call both servers and deal with the first one
func getLiveServer(c config) (*grpc.ClientConn, error) {
	for _, v := range c.servers {
		conn, err := getConnection(v)
		if err == nil {
			return conn, nil
		}
		log.Printf("Unable to dial gRPC server: %v", err)
	}
	return nil, fmt.Errorf("unable to dial either of the gRPC servers")
}

// TODO: Explain this
type sConn struct {
	conn *grpc.ClientConn
	err  error
}

func getAllServers(c config) []sConn {
	var connections []sConn
	for _, v := range c.servers {
		log.Printf("Attempting to get connection to %s\n", v)
		conn, err := getConnection(v)
		connections = append(connections, sConn{
			conn: conn,
			err:  err,
		})
	}
	return connections
}

// allCurrent will attempt to get current values from all servers.
// Will attempt to set tweet bit on all.
// End result is that if single error, continue, if all error, error.
// Hopefully update all, but return a single response from whichever server is live
func allCurrent(c config) ([]tweet, error) {
	log.Println("Running allCurrent")

	connections := getAllServers(c)

	type tweetErr struct {
		tweets []tweet
		err    error
	}

	var res []tweetErr

	// Connect to all servers, get current, work out tweets, and update tweet bits
	for i, v := range connections {
		if v.err == nil {
			log.Printf("Connecting to server %d at %v\n", i+1, v.conn.Target())
			tw, err := current(bpb.NewBgpInfoClient(v.conn), c.dryRun)
			res = append(res, tweetErr{tweets: tw, err: err})
		}
	}

	// Return the first good response. Most of the time this will be the first server in the list.
	for _, v := range res {
		if v.err == nil {
			return v.tweets, v.err
		}
	}

	// This should only execute if none if the configured servers actually gave a response.
	return nil, fmt.Errorf("neither server gave a response for current")
}

// current grabs the current v4 and v6 table count for tweeting.
func current(bgp bpb.BgpInfoClient, dryrun bool) ([]tweet, error) {
	log.Println("Running current")
	counts, err := bgp.GetPrefixCount(context.Background(), &bpb.Empty{})
	if err != nil {
		return nil, err
	}

	// Check for sane IP values
	if counts.GetActive_4() < minV4 {
		return nil, fmt.Errorf("IPv4 count is %d, which is less than the minimum sane value of %d",
			counts.GetActive_4(), minV4)
	}
	if counts.GetActive_6() < minV6 {
		return nil, fmt.Errorf("IPv6 count is %d, which is less than the minimum sane value of %d",
			counts.GetActive_6(), minV6)
	}

	suffixes := []func(b bpb.BgpInfoClient, counts *bpb.PrefixCountResponse) (string, string, error){
		func(b bpb.BgpInfoClient, counts *bpb.PrefixCountResponse) (string, string, error) {
			return asnMostBlocks(context.Background(), bgp, counts)
		},
		func(b bpb.BgpInfoClient, counts *bpb.PrefixCountResponse) (string, string, error) {
			return asnMostInvalids(context.Background(), bgp, counts)
		},
		func(b bpb.BgpInfoClient, counts *bpb.PrefixCountResponse) (string, string, error) {
			return largeSubnetPercentage(context.Background(), bgp, counts)
		},
		// TODO: This is very compute heavy
		//func(b bpb.BgpInfoClient) (string, error) {
		//	return b.asnRPKIALLAnnounced(context.Background(), &bpb.Empty{})
		//},
	}
	// Seed the random number generator
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// Generate a random index
	randomIndex := r.Intn(len(suffixes))
	// Call the randomly selected function and return the result
	v4, v6, err := suffixes[randomIndex](bgp, counts)
	if err != nil {
		return nil, err
	}

	v4Tweet := tweet{
		account: "bgp4table",
		message: v4,
	}
	v6Tweet := tweet{
		account: "bgp6table",
		message: v6,
	}

	if err := setTweetBit(bgp, counts.GetTime(), dryrun); err != nil {
		log.Printf("Unable to set tweet bit, but continuing on: %v", err)
	}

	log.Printf("IPv4: %s\nIPv6: %s\n", v4, v6)
	return []tweet{v4Tweet, v6Tweet}, nil
}

func prefixCurrent(_ bpb.BgpInfoClient, counts *bpb.PrefixCountResponse) (*strings.Builder, *strings.Builder, error) {
	// Calculate deltas.
	v4DeltaH := int(counts.GetActive_4()) - int(counts.GetSixhoursv4())
	v6DeltaH := int(counts.GetActive_6()) - int(counts.GetSixhoursv6())
	v4DeltaW := int(counts.GetActive_4()) - int(counts.GetWeekagov4())
	v6DeltaW := int(counts.GetActive_6()) - int(counts.GetWeekagov6())

	// Formulate updates
	var v4Update, v6Update strings.Builder
	v4Update.WriteString(fmt.Sprintf("I see %d IPv4 prefixes. ", counts.GetActive_4()))
	v4Update.WriteString(deltaMessage(v4DeltaH, v4DeltaW))

	v6Update.WriteString(fmt.Sprintf("I see %d IPv6 prefixes. ", counts.GetActive_6()))
	v6Update.WriteString(deltaMessage(v6DeltaH, v6DeltaW))

	return &v4Update, &v6Update, nil
}

func asnMostBlocks(_ context.Context, bgp bpb.BgpInfoClient, counts *bpb.PrefixCountResponse) (string, string, error) {
	v4, v6, err := prefixCurrent(bgp, counts)
	if err != nil {
		return "", "", err
	}
	v4.WriteString(" This has asMostBlocks")
	v6.WriteString(" This has asMostBlocks")
	return v4.String(), v6.String(), nil
}

func asnMostInvalids(_ context.Context, bgp bpb.BgpInfoClient, counts *bpb.PrefixCountResponse) (string, string, error) {
	v4, v6, err := prefixCurrent(bgp, counts)
	if err != nil {
		return "", "", err
	}
	v4.WriteString(" This has asMostInvalids")
	v6.WriteString(" This has asMostInvalids")
	return v4.String(), v6.String(), nil
}

func largeSubnetPercentage(_ context.Context, bgp bpb.BgpInfoClient, counts *bpb.PrefixCountResponse) (string, string, error) {
	v4, v6, err := prefixCurrent(bgp, counts)
	if err != nil {
		return "", "", err
	}
	// Calculate large subnets percentages
	percentV4 := float32(counts.GetSlash24()) / float32(counts.GetActive_4()) * 100
	percentV6 := float32(counts.GetSlash48()) / float32(counts.GetActive_6()) * 100

	v4.WriteString(fmt.Sprintf(" %.2f%% of prefixes are /24.", percentV4))
	v6.WriteString(fmt.Sprintf(" %.2f%% of prefixes are /48.", percentV6))

	return v4.String(), v6.String(), nil
}

// deltaMessage creates the update message itself. Uses the deltas to formulate the exact message.
func deltaMessage(h, w int) string {
	log.Println("Running deltaMessage")
	var update strings.Builder
	switch {
	case h == 1:
		update.WriteString("This is 1 more prefix than 6 hours ago ")
	case h == -1:
		update.WriteString("This is 1 less prefix than 6 hours ago ")
	case h < 0:
		update.WriteString(fmt.Sprintf("This is %d fewer prefixes than 6 hours ago ", -h))
	case h > 0:
		update.WriteString(fmt.Sprintf("This is %d more prefixes than 6 hours ago ", h))
	default:
		update.WriteString("No change in the amount of prefixes from 6 hours ago ")

	}

	switch {
	case w == 1:
		update.WriteString("and 1 more than a week ago.")
	case w == -1:
		update.WriteString("and 1 less than a week ago.")
	case w < 0:
		update.WriteString(fmt.Sprintf("and %d fewer than a week ago.", -w))
	case w > 0:
		update.WriteString(fmt.Sprintf("and %d more than a week ago.", w))
	default:
		update.WriteString("and no change in the amount from a week ago.")

	}

	return update.String()
}

func setTweetBit(cpb bpb.BgpInfoClient, time uint64, dryrun bool) error {
	log.Println("Running setTweetBit")

	if dryrun {
		log.Printf("dry run set, so not setting tweet bit")
		return nil
	}

	timestamp := &bpb.Timestamp{
		Time: time,
	}
	_, err := cpb.UpdateTweetBit(context.Background(), timestamp)
	if err != nil {
		return fmt.Errorf("error: received error when trying to set tweet bit")
	}
	return nil
}

func subnets(c config) ([]tweet, error) {
	log.Println("Running subnets")

	conn, err := getLiveServer(c)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	cpb := bpb.NewBgpInfoClient(conn)
	pieData, err := cpb.GetPieSubnets(context.Background(), &bpb.Empty{})
	if err != nil {
		log.Fatalf("Unable to send proto: %s", err)
	}

	v4Colours := []string{"burlywood", "lightgreen", "lightskyblue", "lightcoral", "gold"}
	v6Colours := []string{"lightgreen", "burlywood", "lightskyblue", "violet", "linen", "lightcoral", "gold"}
	v4Labels := []string{"/19-/21", "/16-/18", "/22", "/23", "/24"}
	v6Labels := []string{"/32", "/44", "/40", "/36", "/29", "The Rest", "/48"}

	t := time.Now()
	v4Meta := &gpb.Metadata{
		Title:   fmt.Sprintf("Current prefix range distribution for IPv4 (%s)", t.Format("02-Jan-2006")),
		XAxis:   uint32(12),
		YAxis:   uint32(10),
		Colours: v4Colours,
		Labels:  v4Labels,
	}
	v6Meta := &gpb.Metadata{
		Title:   fmt.Sprintf("Current prefix range distribution for IPv6 (%s)", t.Format("02-Jan-2006")),
		XAxis:   uint32(12),
		YAxis:   uint32(10),
		Colours: v6Colours,
		Labels:  v6Labels,
	}

	v4Subnets := []uint32{
		pieData.GetMasks().GetV4_19() + pieData.GetMasks().GetV4_20() + pieData.GetMasks().GetV4_21(),
		pieData.GetMasks().GetV4_16() + pieData.GetMasks().GetV4_17() + pieData.GetMasks().GetV4_18(),
		pieData.GetMasks().GetV4_22(),
		pieData.GetMasks().GetV4_23(),
		pieData.GetMasks().GetV4_24(),
	}
	v6Subnets := []uint32{
		pieData.GetMasks().GetV6_32(),
		pieData.GetMasks().GetV6_44(),
		pieData.GetMasks().GetV6_40(),
		pieData.GetMasks().GetV6_36(),
		pieData.GetMasks().GetV6_29(),
		pieData.GetV6Total() - pieData.GetMasks().GetV6_32() - pieData.GetMasks().GetV6_44() -
			pieData.GetMasks().GetV6_40() - pieData.GetMasks().GetV6_36() - pieData.GetMasks().GetV6_29() -
			pieData.GetMasks().GetV6_48(),
		pieData.GetMasks().GetV6_48(),
	}

	// Dial the grapher to retrieve graphs via matplotlib
	// TODO: Is this not too much stuff in a single function?
	req := &gpb.PieChartRequest{
		Metadatas: []*gpb.Metadata{v4Meta, v6Meta},
		Subnets: &gpb.SubnetFamily{
			V4Values: v4Subnets,
			V6Values: v6Subnets,
		},
		Copyright: "data by Darren O'Connor | www.mellowd.dev",
	}

	grp, err := getTLSConnection(c.grapher)
	if err != nil {
		return nil, err
	}
	defer grp.Close()

	gpb := gpb.NewGrapherClient(grp)

	resp, err := gpb.GetPieChart(context.Background(), req)
	if err != nil {
		return nil, err
	}

	// There should be two images, if not something's gone wrong.
	if len(resp.GetImages()) < 2 {
		return nil, fmt.Errorf("less than two images returned")
	}

	v4Tweet := tweet{
		account: "bgp4table",
		message: v4Meta.Title,
		media:   resp.GetImages()[0].GetImage(),
	}
	v6Tweet := tweet{
		account: "bgp6table",
		message: v6Meta.Title,
		media:   resp.GetImages()[1].GetImage(),
	}

	return []tweet{v4Tweet, v6Tweet}, nil
}

// TODO: Remove outliers from graphs because it's looks rubbish!
func movement(c config, p bpb.MovementRequest_TimePeriod) ([]tweet, error) {
	log.Println("Running movement")

	// Get yesterday's date
	y := time.Now().AddDate(0, 0, -1)

	conn, err := getLiveServer(c)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	cpb := bpb.NewBgpInfoClient(conn)
	graphData, err := cpb.GetMovementTotals(context.Background(), &bpb.MovementRequest{Period: p})
	if err != nil {
		return nil, err
	}

	// Determine image title and update message depending on time period given.
	var period string
	var message string
	switch p {
	case bpb.MovementRequest_WEEK:
		period = "week"
		message = "Weekly BGP table movement #BGP"
	case bpb.MovementRequest_MONTH:
		period = "month"
		message = "Monthly BGP table movement #BGP"
	case bpb.MovementRequest_SIXMONTH:
		period = "6 months"
		message = "BGP table movement for the last 6 months #BGP"
	case bpb.MovementRequest_ANNUAL:
		period = "year"
		message = "Annual BGP table movement #BGP"
	default:
		return nil, fmt.Errorf("time Period not set")
	}

	// metadata to create images
	v4Meta := &gpb.Metadata{
		Title:  fmt.Sprintf("IPv4 table movement for %s ending %s", period, y.Format("02-Jan-2006")),
		XAxis:  uint32(12),
		YAxis:  uint32(10),
		Colour: "#238341",
	}
	v6Meta := &gpb.Metadata{
		Title:  fmt.Sprintf("IPv6 table movement for %s ending %s", period, y.Format("02-Jan-2006")),
		XAxis:  uint32(12),
		YAxis:  uint32(10),
		Colour: "#0041A0",
	}

	// repack counts and dates to grapher proto format.
	tt := []*gpb.TotalTime{}
	for _, i := range graphData.GetValues() {
		tt = append(tt, &gpb.TotalTime{
			V4Values: i.GetV4Values(),
			V6Values: i.GetV6Values(),
			Time:     i.GetTime(),
		})
	}
	req := &gpb.LineGraphRequest{
		Metadatas:  []*gpb.Metadata{v4Meta, v6Meta},
		TotalsTime: tt,
		Copyright:  "data by daz.bgpstuff.net | www.mellowd.dev",
	}

	// Dial the grapher to retrive graphs via matplotlib
	// TODO: seperate this?
	grp, err := getTLSConnection(c.grapher)
	if err != nil {
		return nil, err
	}
	defer grp.Close()
	gpb := gpb.NewGrapherClient(grp)

	resp, err := gpb.GetLineGraph(context.Background(), req)
	if err != nil {
		return nil, err
	}

	// There should be two images, if not something's gone wrong.
	if len(resp.GetImages()) < 2 {
		return nil, fmt.Errorf("less than two images returned")
	}

	v4Tweet := tweet{
		account: "bgp4table",
		message: message,
		media:   resp.GetImages()[0].GetImage(),
	}
	v6Tweet := tweet{
		account: "bgp6table",
		message: message,
		media:   resp.GetImages()[1].GetImage(),
	}

	return []tweet{v4Tweet, v6Tweet}, nil
}

func rpki(c config) ([]tweet, error) {
	log.Println("Running rpki")

	conn, err := getLiveServer(c)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	cpb := bpb.NewBgpInfoClient(conn)

	rpkiData, err := cpb.GetRpki(context.Background(), &bpb.Empty{})
	if err != nil {
		return nil, err
	}

	// metadata to create images
	v4Meta := &gpb.Metadata{
		Title: fmt.Sprintf("Current RPKI status for IPv4 (%s)", time.Now().Format("02-Jan-2006")),
		XAxis: uint32(12),
		YAxis: uint32(10),
	}
	v6Meta := &gpb.Metadata{
		Title: fmt.Sprintf("Current RPKI status for IPv6 (%s)", time.Now().Format("02-Jan-2006")),
		XAxis: uint32(12),
		YAxis: uint32(10),
	}

	// repack
	// TODO: Can I have messages defined in a common way?
	rpkis := &gpb.RPKI{
		V4Valid:   rpkiData.GetV4Valid(),
		V4Invalid: rpkiData.GetV4Invalid(),
		V4Unknown: rpkiData.GetV4Unknown(),
		V6Valid:   rpkiData.GetV6Valid(),
		V6Invalid: rpkiData.GetV6Invalid(),
		V6Unknown: rpkiData.GetV6Unknown(),
	}

	req := &gpb.RPKIRequest{
		Metadatas: []*gpb.Metadata{v4Meta, v6Meta},
		Rpkis:     rpkis,
		Copyright: "data by daz.bgpstuff.net/ | www.mellowd.dev",
	}

	// Dial the grapher to retrive graphs via matplotlib
	grp, err := getTLSConnection(c.grapher)
	if err != nil {
		return nil, err
	}
	defer grp.Close()
	gpb := gpb.NewGrapherClient(grp)

	resp, err := gpb.GetRPKI(context.Background(), req)
	if err != nil {
		return nil, err
	}

	// There should be two images, if not something's gone wrong.
	if len(resp.GetImages()) < 2 {
		return nil, fmt.Errorf("less than two images returned")
	}

	v4Tweet := tweet{
		account: "bgp4table",
		message: "Current RPKI status IPv4 #RPKI",
		media:   resp.GetImages()[0].GetImage(),
	}
	v6Tweet := tweet{
		account: "bgp6table",
		message: "Current RPKI status IPv6 #RPKI",
		media:   resp.GetImages()[1].GetImage(),
	}

	return []tweet{v4Tweet, v6Tweet}, nil
}

func postTweet(t tweet, cf *ini.File) error {

	in := &gotwi.NewClientInput{
		AuthenticationMethod: gotwi.AuthenMethodOAuth1UserContext,
		OAuthToken:           cf.Section(t.account).Key("accessToken").String(),
		OAuthTokenSecret:     cf.Section(t.account).Key("accessSecret").String(),
		APIKey:               cf.Section(t.account).Key("apiKey").String(),
		APISecret:            cf.Section(t.account).Key("apiSecret").String(),
	}
	api := anaconda.NewTwitterApiWithCredentials(
		cf.Section(t.account).Key("accessToken").String(),
		cf.Section(t.account).Key("accessSecret").String(),
		cf.Section(t.account).Key("apiKey").String(),
		cf.Section(t.account).Key("apiSecret").String(),
	)

	c, err := gotwi.NewClient(in)
	if err != nil {
		return err
	}

	p := &types.CreateInput{
		Text: gotwi.String(t.message),
	}

	// Images need to be uploaded and referred to in an actual tweet
	var media anaconda.Media
	if t.media != nil {
		// TODO: Why am I ignoring errors here?
		media, _ = api.UploadMedia(base64.StdEncoding.EncodeToString(t.media))
		p.Media = &types.CreateInputMedia{
			MediaIDs: []string{media.MediaIDString},
		}
	}

	/*
		// Videos are a lot more complicated
		// https://developer.twitter.com/en/docs/tutorials/uploading-media
		// Note: This only deals with small files
		// TODO: Make this deal with larger files too!
		if t.video != nil {
			size := len(t.video)
			// Step 1 - Init
			// TODO: Why am I ignoring errors here?
			cm, _ := api.UploadVideoInit(size, "video/mp4")

			// Step 2 -- Append
			encoded := base64.StdEncoding.EncodeToString(t.video)
			// TODO: check for errors
			api.UploadVideoAppend(cm.MediaIDString, 0, encoded)

			// Step 3 - Finalise
			// TODO: check for errors
			api.UploadVideoFinalize(cm.MediaIDString)

			// attach
			v.Set("media_ids", cm.MediaIDString)

		}
	*/

	// post it!
	_, err = managetweet.Create(context.TODO(), c, p)
	if err != nil {
		return fmt.Errorf("error: unable to post tweet %v", err)
	}
	return nil
}

func postToot(t tweet, cf *ini.File) error {
	ctx := context.TODO()
	// read mastodon account credentials
	account := "bgp6mastodon"
	if t.account == "bgp4table" {
		account = "bgp4mastodon"
	}
	server := cf.Section(account).Key("server").String()
	clientID := cf.Section(account).Key("clientID").String()
	clientSecret := cf.Section(account).Key("clientSecret").String()
	accessToken := cf.Section(account).Key("accessToken").String()
	email := cf.Section(account).Key("email").String()
	password := cf.Section(account).Key("password").String()

	// set up mastodon client
	c := mastodon.NewClient(&mastodon.Config{
		Server:       server,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AccessToken:  accessToken,
	})

	// authenticate client
	if err := c.Authenticate(ctx, email, password); err != nil {
		return err
	}

	toot := mastodon.Toot{}
	toot.Status = t.message

	// Images need to be uploaded and referred to in an actual toot
	if t.media != nil {
		// TODO: change this to uploadmediafrombytes once upgraded
		att, err := c.UploadMediaFromReader(ctx, bytes.NewReader(t.media))
		if err != nil {
			return err
		}
		toot.MediaIDs = append(toot.MediaIDs, att.ID)
	}

	// post it!
	if _, err := c.PostStatus(ctx, &toot); err != nil {
		return fmt.Errorf("error: unable to post toot %v", err)
	}

	return nil
}

func postBsky(t tweet, cf *ini.File) error {
	user := cf.Section("bsky").Key("username").String()
	hand := cf.Section("bsky").Key("handle").String()
	pass := cf.Section("bsky").Key("password").String()

	c := bsky.NewClient(
		bsky.Account{
			Username: user,
			Handle:   hand,
			Password: pass,
		},
	)

	// Authenticate
	token, err := c.Authenticate()
	if err != nil {
		return err
	}
	did, err := c.GetDID(token, hand)

	if t.media != nil {
		// Upload image
		ul, err := c.UploadImage(token, t.media)
		if err != nil {
			return err
		}
		post := bsky.ImagePostContent{
			Type:      "app.bsky.feed.post",
			Text:      t.message,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		post.Embed = bsky.PostEmbed{
			Type: "app.bsky.embed.images",
			Images: []bsky.EmbedImage{
				{
					Image: bsky.Blob{
						Type:     "blob",
						Ref:      bsky.BlobRef{Link: ul.Ref},
						MimeType: fmt.Sprintf("image/%s", ul.Fmt),
						Size:     len(t.media),
					},
					AspectRatio: bsky.AspectRatio{Width: ul.Cfg.Width, Height: ul.Cfg.Height},
				},
			},
		}
		// Post it
		if err := c.CreatePost(token, did, post); err != nil {
			return err
		}
		return nil
	}

	post := bsky.TextPostContent{
		Type:      "app.bsky.feed.post",
		Text:      t.message,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	// Post it
	if err := c.CreatePost(token, did, post); err != nil {
		return err
	}
	return nil
}
