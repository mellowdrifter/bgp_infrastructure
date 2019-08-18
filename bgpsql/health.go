// health is used when checking if local server is primary or not.
// If any of these tests fail, the local server cannot be primary
package main

import (
	"database/sql"
	"log"
	"os/exec"
)

//TODO: This does nothing right now!

func isHealthy() bool {
	//return dbHealth() && birdRunning() && minCount() && minPeers()
	return true
}

// Can we ping the datbase
func dbHealth(db *sql.DB) bool {
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
	cmd := exec.Command("pgrep", "bird$")
	out, _ := cmd.CombinedOutput()
	if len(out) > 0 {
		log.Printf("bird4 is running")
		b4 = true
	}
	cmd = exec.Command("pgrep", "bird6$")
	out, _ = cmd.CombinedOutput()
	if len(out) > 0 {
		log.Printf("bird6 is running")
		b6 = true
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
