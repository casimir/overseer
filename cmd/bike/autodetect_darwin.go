package main

import (
	"os/exec"
	"regexp"
)

func getSsid() (string, bool) {
	re := regexp.MustCompile(`(?m)^\s+SSID: (.+)$`)
	out, err := exec.Command("/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport", "-I").Output()
	if err != nil {
		return "", false
	}
	if matches := re.FindStringSubmatch(string(out)); matches != nil {
		return matches[1], true
	}
	return "", false
}
