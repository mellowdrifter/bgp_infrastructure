package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/mellowdrifter/bgp_infrastructure/internal/xapi"
	"gopkg.in/ini.v1"
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

	client := xapi.NewClient(
		cf.Section("x_tester").Key("consumer_key").String(),
		cf.Section("x_tester").Key("consumer_secret").String(),
		cf.Section("x_tester").Key("access_token").String(),
		cf.Section("x_tester").Key("access_secret").String(),
	)

	// Upload image
	mediaID, err := client.UploadImage("test.png")
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	// Post tweet
	tweetID, err := client.PostTweet("x API test, text only")
	if err != nil {
		log.Fatalf("Tweet failed: %v", err)
	}
	fmt.Printf("Tweet posted successfully! ID: %s\n", tweetID)

	// Post tweet with media
	tweetID, err = client.PostTweet("x API test with media", mediaID)
	if err != nil {
		log.Fatalf("Tweet failed: %v", err)
	}
	fmt.Printf("Tweet posted successfully! ID: %s\n", tweetID)
}
