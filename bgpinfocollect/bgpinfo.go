package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
)

func main() {
	out := getTableTotal()
	fmt.Printf("Totals: %q\n", out)
	fmt.Printf("RIB: %s, FIB: %s\n", string(out[0]), string(out[1]))

}

func getOutput(cmd string) ([]byte, error) {
	fmt.Printf("Running getOutput with cmd %s\n", cmd)
	cmdOut, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return cmdOut, err
	}

	return cmdOut, err
}

func getTableTotal() [][]byte {
	cmd := "/usr/sbin/birdc show route count | grep routes | awk {'print $3, $6'}"
	v4, err := getOutput(cmd)
	if err != nil {
		log.Fatal(err)
	}
	return bytes.Fields(v4)
}
