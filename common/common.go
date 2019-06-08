package common

import (
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// GetOutput is a helper function to run commands and return outputs to other functions.
func GetOutput(cmd string) (string, error) {
	log.Printf("Running getOutput with cmd %s\n", cmd)
	cmdOut, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return string(cmdOut), err
	}

	return strings.TrimSuffix(string(cmdOut), "\n"), err
}

// StringToUint32 is a helper function as many times I need to do this conversion.
func StringToUint32(s string) uint32 {
	val, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("Can't convert to integer: %s", err)
	}
	return uint32(val)
}

// SetListOfStrings returns a slice of strings with no duplicates.
// Go has no built-in set function, so doing it here
func SetListOfStrings(input []string) []string {
	u := make([]string, 0, len(input))
	m := make(map[string]bool)
	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = true
			u = append(u, val)
		}
	}
	return u

}

// InFirstButNotSecond returns the second slice subtracted from the first.
// Go has no built-in function to do this.
func InFirstButNotSecond(first, second []string) []string {
	var third []string
	iterMap := make(map[string]bool)
	for _, element := range second {
		iterMap[element] = true
	}
	for _, element := range first {
		if _, ok := iterMap[element]; !ok {
			third = append(third, element)
		}
	}
	return third
}

// Intersection returns a slice that is an intersection of two slices.
// Go has no built-in function to do this.
func Intersection(first, second []string) []string {
	var third []string
	iterMap := make(map[string]bool)
	for _, element := range second {
		iterMap[element] = true
	}
	for _, element := range first {
		if _, ok := iterMap[element]; ok {
			third = append(third, element)
		}
	}
	return third
}

// TimeFunction logs total time to execute a function.
func TimeFunction(start time.Time, name string) {
	log.Printf("%s took %s\n", name, time.Since(start))
}
