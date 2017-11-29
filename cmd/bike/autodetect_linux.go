package main

import (
	"os/exec"
	"regexp"
)

func getSsid() (string, bool) {
	re := regexp.MustCompile(`(?m)^wl.+\s+ESSID:"(.+)"`)
	out, err := exec.Command("iwconfig").Output()
	if err != nil {
		return "", false
	}
	if matches := re.FindStringSubmatch(string(out)); matches != nil {
		return matches[1], true
	}
	return "", false
}
