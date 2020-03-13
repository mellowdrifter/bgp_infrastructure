package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ChimeraCoder/anaconda"
	bpb "github.com/mellowdrifter/bgp_infrastructure/proto/bgpsql"
	gpb "github.com/mellowdrifter/bgp_infrastructure/proto/grapher"
	"google.golang.org/grpc"
	"gopkg.in/ini.v1"
)

type tweet struct {
	account string
	message string
	media   []byte
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

// Pull out most of the intial set up into a separate function
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

	config.log = cf.Section("log").Key("log").String()
	config.grapher = cf.Section("grapher").Key("server").String()
	config.servers = cf.Section("bgpinfo").Key("server").ValueWithShadows()

	// What action are we going to do.
	config.action = flag.String("action", "", "an action to perform")
	config.time = flag.String("time", "", "a time period")
	flag.Parse()

	return config, nil

}

func main() {

	config, err := setup()
	if err != nil {
		log.Fatal(err)
	}

	// Set up log file
	f, err := os.OpenFile(config.log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open logfile: %v\n", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// App only does a single action at a time.
	var tweets []tweet
	switch *config.action {
	case "current":
		tweets, err = allCurrent(config)
	case "subnets":
		tweets, err = subnets(config)
	case "rpki":
		tweets, err = rpki(config)
	case "movement":
		var period bpb.MovementRequest_TimePeriod
		switch *config.time {
		case "week":
			period = bpb.MovementRequest_WEEK
		case "month":
			period = bpb.MovementRequest_MONTH
		case "month6":
			period = bpb.MovementRequest_SIXMONTH
		case "annual":
			period = bpb.MovementRequest_ANNUAL
		default:
			log.Fatalf("Movement request requires a time period")
		}
		tweets, err = movement(config, period)
	default:
		log.Fatalf("At least one action must be specified")
	}
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	// Post tweets.
	// TODO: Have a dry-run to print and save images locally
	for _, tweet := range tweets {
		if err := postTweet(tweet, config.file); err != nil {
			log.Fatal(err)
		}
	}
}

// getConnection will return a connection to a gRPC server. Caller should close.
func getConnection(srv string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(srv, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("unable to dial gRPC server: %v", err)
	}
	return conn, err

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
	var conns []sConn
	for _, v := range c.servers {
		log.Printf("Attempting to get connection to %s\n", v)
		conn, err := getConnection(v)
		conns = append(conns, sConn{
			conn: conn,
			err:  err,
		})
	}
	return conns

}

// allCurrent will attempt to get current values from all servers.
// Will attempt to set tweet bit on all.
// End result is that if single error, continue, if all error, error.
// Hopefully update all, but return a single response from whichever server is live
func allCurrent(c config) ([]tweet, error) {
	log.Println("Running allCurrent")

	conns := getAllServers(c)

	type tweetErr struct {
		tweets []tweet
		err    error
	}

	var res []tweetErr

	// Connect to all servers, get current, work out tweets, and update tweet bits
	for i, v := range conns {
		if v.err == nil {
			log.Printf("Connecting to server %d at %v\n", i, v.conn.Target())
			tw, err := current(bpb.NewBgpInfoClient(v.conn))
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
func current(b bpb.BgpInfoClient) ([]tweet, error) {

	log.Println("Running current")
	counts, err := b.GetPrefixCount(context.Background(), &bpb.Empty{})
	if err != nil {
		return nil, err
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

	if err := setTweetBit(b, counts.GetTime()); err != nil {
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

func setTweetBit(cpb bpb.BgpInfoClient, time uint64) error {
	log.Println("Running setTweetBit")

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
	v4Lables := []string{"/19-/21", "/16-/18", "/22", "/23", "/24"}
	v6Lables := []string{"/32", "/44", "/40", "/36", "/29", "The Rest", "/48"}

	t := time.Now()
	v4Meta := &gpb.Metadata{
		Title:   fmt.Sprintf("Current prefix range distribution for IPv4 (%s)", t.Format("02-Jan-2006")),
		XAxis:   uint32(12),
		YAxis:   uint32(10),
		Colours: v4Colours,
		Labels:  v4Lables,
	}
	v6Meta := &gpb.Metadata{
		Title:   fmt.Sprintf("Current prefix range distribution for IPv6 (%s)", t.Format("02-Jan-2006")),
		XAxis:   uint32(12),
		YAxis:   uint32(10),
		Colours: v6Colours,
		Labels:  v6Lables,
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

	// Dial the grapher to retrive graphs via matplotlib
	// TODO: IS this not too much stuff in a single function?
	req := &gpb.PieChartRequest{
		Metadatas: []*gpb.Metadata{v4Meta, v6Meta},
		Subnets: &gpb.SubnetFamily{
			V4Values: v4Subnets,
			V6Values: v6Subnets,
		},
		Copyright: "data by @mellowdrifter | www.mellowd.dev",
	}
	grp, err := getConnection(c.grapher)
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
	grp, err := getConnection(c.grapher)
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
	grp, err := getConnection(c.grapher)
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
