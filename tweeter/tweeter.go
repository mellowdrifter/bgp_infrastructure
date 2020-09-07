package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	bpb "github.com/mellowdrifter/bgp_infrastructure/tweeter/proto/bgpsql"
	gpb "github.com/mellowdrifter/bgp_infrastructure/tweeter/proto/grapher"

	"github.com/ChimeraCoder/anaconda"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/ini.v1"
)

const (
	// If I see IPv4 and IPv6 values less than these values, there is an issue.
	// This value can be revised once every 6 months or so.
	minV4 = 800000
	minV6 = 80000
)

type tweet struct {
	account string
	message string
	media   []byte
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
}

type config struct {
	log     string
	grapher string
	action  *string
	time    *string
	servers []string
	file    *ini.File
	dryRun  bool
}

type tweeter struct {
	mux *http.ServeMux
	mu  sync.Mutex
	cfg config
}

// Pull out most of the initial set up into a separate function
func setup() (config, error) {
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

	config.grapher = cf.Section("grapher").Key("server").String()
	config.servers = cf.Section("bgpinfo").Key("server").ValueWithShadows()

	flag.Parse()

	return config, nil

}

// Cloud Run should use this.
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg, err := setup()
	if err != nil {
		log.Fatalf("unable to set things up: %v", err)
	}

	var srv tweeter
	srv.mux = http.NewServeMux()
	srv.cfg = cfg

	srv.mux.HandleFunc("/post", srv.post())
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
func faviconHandler(w http.ResponseWriter, r *http.Request) {
	return
}

// Basic index for now.
func (t *tweeter) dryrun() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("This is the full request: %#v\n", r)
		log.Printf("url is %v\n", r.RequestURI)
		t.mu.Lock()
		defer t.mu.Unlock()
		t.cfg.dryRun = true

		todo := whatToTweet(time.Now())
		//TEMP
		todo.rpkiPie = true
		todo.subnetPie = true
		todo.weekGraph = true
		todo.monthGraph = true
		todo.sixMonthGraph = true
		todo.annualGraph = true
		// TEMP

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
			// Post tweets.
			if err := postTweet(tweet, t.cfg.file); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Printf("error when posting tweet: %v", err)
			}
		}
	}
}

// getTweets will compile all tweets as according to the todo list of tweets.
func getTweets(todo toTweet, cfg config) ([]tweet, error) {
	var listOfTweets []tweet

	if todo.tableSize {
		tweets, err := allCurrent(cfg)
		if err != nil {
			return listOfTweets, fmt.Errorf("Unable to gather table size info: %v", err)
		}
		listOfTweets = append(listOfTweets, tweets...)
	}

	if todo.weekGraph {
		tweets, err := movement(cfg, bpb.MovementRequest_WEEK)
		if err != nil {
			return listOfTweets, fmt.Errorf("Unable to gather weekly graph info: %v", err)
		}
		listOfTweets = append(listOfTweets, tweets...)
	}
	if todo.monthGraph {
		tweets, err := movement(cfg, bpb.MovementRequest_MONTH)
		if err != nil {
			return listOfTweets, fmt.Errorf("Unable to gather monthly graph info: %v", err)
		}
		listOfTweets = append(listOfTweets, tweets...)
	}
	if todo.sixMonthGraph {
		tweets, err := movement(cfg, bpb.MovementRequest_SIXMONTH)
		if err != nil {
			return listOfTweets, fmt.Errorf("Unable to gather six monthly graph info: %v", err)
		}
		listOfTweets = append(listOfTweets, tweets...)
	}
	if todo.annualGraph {
		tweets, err := movement(cfg, bpb.MovementRequest_ANNUAL)
		if err != nil {
			return listOfTweets, fmt.Errorf("Unable to gather six monthly graph info: %v", err)
		}
		listOfTweets = append(listOfTweets, tweets...)
	}

	if todo.rpkiPie {
		tweets, err := rpki(cfg)
		if err != nil {
			return listOfTweets, fmt.Errorf("Unable to generate RPKI tweets: %v", err)
		}
		listOfTweets = append(listOfTweets, tweets...)
	}

	if todo.subnetPie {
		tweets, err := subnets(cfg)
		if err != nil {
			return listOfTweets, fmt.Errorf("Unable to generate subnet pie tweets: %v", err)
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
	todo.monthGraph = (now.Day() == 1)

	// 1st of July also tweet a 6 month growth graph.
	todo.sixMonthGraph = (now.Day() == 1 && now.Month() == time.July)

	// Annual graph. Post on 3rd of January as no-one is around 1st and 2nd.
	todo.annualGraph = (now.Day() == 3 && now.Month() == time.January)

	// On Wednesday I tweet the subnet pie graph.
	todo.subnetPie = (now.Weekday() == time.Wednesday)

	// On Thursday I tweet the RPKI status.
	todo.rpkiPie = (now.Weekday() == time.Thursday)

	return todo
}

func run() {

	/*
	 */
}

// getConnection will return a connection to a gRPC server. Caller should close.
// TODO: Do the funky thing where you return the closer.
func getConnection(srv string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(srv, grpc.WithInsecure())
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
	return nil, fmt.Errorf("Neither server gave a response for current")
}

// current grabs the current v4 and v6 table count for tweeting.
func current(b bpb.BgpInfoClient, dryrun bool) ([]tweet, error) {

	log.Println("Running current")
	counts, err := b.GetPrefixCount(context.Background(), &bpb.Empty{})
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

	// Calculate deltas.
	v4DeltaH := int(counts.GetActive_4()) - int(counts.GetSixhoursv4())
	v6DeltaH := int(counts.GetActive_6()) - int(counts.GetSixhoursv6())
	v4DeltaW := int(counts.GetActive_4()) - int(counts.GetWeekagov4())
	v6DeltaW := int(counts.GetActive_6()) - int(counts.GetWeekagov6())

	// Calculate large subnets percentages
	percentV4 := float32(counts.GetSlash24()) / float32(counts.GetActive_4()) * 100
	percentV6 := float32(counts.GetSlash48()) / float32(counts.GetActive_6()) * 100

	// Formulate updates
	var v4Update, v6Update strings.Builder
	v4Update.WriteString(fmt.Sprintf("I see %d IPv4 prefixes. ", counts.GetActive_4()))
	v4Update.WriteString(deltaMessage(v4DeltaH, v4DeltaW))
	v4Update.WriteString(fmt.Sprintf(". %.2f%% of prefixes are /24.", percentV4))

	v6Update.WriteString(fmt.Sprintf("I see %d IPv6 prefixes. ", counts.GetActive_6()))
	v6Update.WriteString(deltaMessage(v6DeltaH, v6DeltaW))
	v6Update.WriteString(fmt.Sprintf(". %.2f%% of prefixes are /48.", percentV6))

	v4Tweet := tweet{
		account: "bgp4table",
		message: v4Update.String(),
	}
	v6Tweet := tweet{
		account: "bgp6table",
		message: v6Update.String(),
	}

	if err := setTweetBit(b, counts.GetTime(), dryrun); err != nil {
		log.Printf("Unable to set tweet bit, but continuing on: %v", err)
	}
	return []tweet{v4Tweet, v6Tweet}, nil

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
		update.WriteString("and 1 more than a week ago")
	case w == -1:
		update.WriteString("and 1 less than a week ago")
	case w < 0:
		update.WriteString(fmt.Sprintf("and %d fewer than a week ago", -w))
	case w > 0:
		update.WriteString(fmt.Sprintf("and %d more than a week ago", w))
	default:
		update.WriteString("and no change in the amount from a week ago")

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
	defer conn.Close()
	if err != nil {
		return nil, err
	}

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
	// TODO: IS this not too much stuff in a single function?
	req := &gpb.PieChartRequest{
		Metadatas: []*gpb.Metadata{v4Meta, v6Meta},
		Subnets: &gpb.SubnetFamily{
			V4Values: v4Subnets,
			V6Values: v6Subnets,
		},
		Copyright: "data by @mellowdrifter | www.mellowd.dev",
	}

	grp, err := getTLSConnection(c.grapher)
	defer grp.Close()
	gpb := gpb.NewGrapherClient(grp)

	resp, err := gpb.GetPieChart(context.Background(), req)
	if err != nil {
		return nil, err
	}

	// There should be two images, if not something's gone wrong.
	if len(resp.GetImages()) < 2 {
		return nil, fmt.Errorf("Less than two images returned")
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

func movement(c config, p bpb.MovementRequest_TimePeriod) ([]tweet, error) {
	log.Println("Running movement")

	// Get yesterday's date
	y := time.Now().AddDate(0, 0, -1)

	conn, err := getLiveServer(c)
	defer conn.Close()
	if err != nil {
		return nil, err
	}

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
		return nil, fmt.Errorf("Time Period not set")
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
		Copyright:  "data by @mellowdrifter | www.mellowd.dev",
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
		return nil, fmt.Errorf("Less than two images returned")
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
	defer conn.Close()
	if err != nil {
		return nil, err
	}
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
		Copyright: "data by @mellowdrifter | www.mellowd.dev",
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
		return nil, fmt.Errorf("Less than two images returned")
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
	// read account credentials
	consumerKey := cf.Section(t.account).Key("consumerKey").String()
	consumerSecret := cf.Section(t.account).Key("consumerSecret").String()
	accessToken := cf.Section(t.account).Key("accessToken").String()
	accessSecret := cf.Section(t.account).Key("accessSecret").String()

	// set up twitter client
	api := anaconda.NewTwitterApiWithCredentials(accessToken, accessSecret, consumerKey, consumerSecret)

	// Images need to be uploaded and referred to in an actual tweet
	var media anaconda.Media
	v := url.Values{}
	if t.media != nil {
		media, _ = api.UploadMedia(base64.StdEncoding.EncodeToString(t.media))
		v.Set("media_ids", media.MediaIDString)
	}

	// post it!
	if _, err := api.PostTweet(t.message, v); err != nil {
		return fmt.Errorf("error: unable to post tweet %v", err)
	}

	return nil

}
