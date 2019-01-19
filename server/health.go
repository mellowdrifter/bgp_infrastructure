// health is used when checking if local server is primary or not.
// If any of these tests fail, the local server cannot be primary
package main

import (
	"log"
	"net"
)

func isHealthy() bool {
	return dbHealth() && birdRunning() && minCount() && minPeers()
}

// Can we ping the datbase
func dbHealth() bool {
	err := db.Ping()
	if err != nil {
		log.Printf("unable to ping database for healthcheck: %v\n", err)
		return false
	}
	return true
}

// Is bird running
func birdRunning() bool {
	var b4, b6 bool
	_, err := net.Dial("tcp", ":179")
	if err != nil {
		b4 = true
	} else {
		log.Printf("bird is not running")
	}
	_, err = net.Dial("tcp", "[::]179")
	if err != nil {
		b6 = true
	} else {
		log.Printf("bird6 is not running")
	}
	return b4 && b6
}

// Is there at least 650k v4 prefixes and 45k v6 prefixes
func minCount() bool {
	return true
}

// Do I have at least 5 peers up?
func minPeers() bool {
	return true
}
