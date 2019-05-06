package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path"

	"github.com/ChimeraCoder/anaconda"
	"gopkg.in/ini.v1"
)

func main() {

	// Ensure we get the required config
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	path := fmt.Sprintf("%s/config.ini", path.Dir(exe))
	cf, err := ini.Load(path)
	if err != nil {
		log.Fatalf("failed to read config file: %v\n", err)
	}
	tAPIKey := cf.Section("twitter").Key("API_KEY").String()
	tAPIKeySecret := cf.Section("twitter").Key("API_SECRET").String()
	tAccessToken := cf.Section("twitter").Key("ACCESS_TOKEN").String()
	tAccessTokenSecret := cf.Section("twitter").Key("ACCESS_TOKEN_SECRET").String()
	//logfile := fmt.Sprintf(cf.Section("log").Key("file").String())

	// Set up connection to Twitter API
	twitter := anaconda.NewTwitterApiWithCredentials(tAccessToken, tAccessTokenSecret, tAPIKey, tAPIKeySecret)
	fmt.Printf("%+v\n", twitter)

	stream := twitter.UserStream(url.Values{})

	//stream := twitter.PublicStreamFilter(url.Values{
	//"track": []string{"#route", "#origin", "#aspath", "#asname"},
	//"track": []string{"#love"},
	//})
	fmt.Printf("%+v\n", stream)

	defer stream.Stop()

	for v := range stream.C {
		t, ok := v.(anaconda.Tweet)
		if !ok {
			log.Fatalf("received unexpected value of type %T", v)
		}
		fmt.Printf("%s\n", t.Text)
	}

}
