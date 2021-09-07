package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
)

var ubcranges []*net.IPNet

func ParseIPs() {
	ubcranges = make([]*net.IPNet, 0)
	var ubcips []string
	ipsdata, err := ioutil.ReadFile("ubcIPs.json")
	if err != nil {
		log.Panicf("Error reading ubcIPs file: %v\n", err)
	}
	err = json.Unmarshal(ipsdata, &ubcips)
	if err != nil {
		log.Panicf("Error parsing ubcIPs file: %v\n", err)
	}
	// Test Mode, add local IP ranges
	if testmode {
		ubcips = append(ubcips, "192.168.1.1/24")
		ubcips = append(ubcips, "192.168.0.1/24")
		ubcips = append(ubcips, "127.0.0.1/8")
		log.Println("TestMode: Added local network IP ranges")
	}
	// Build ranges
	for _, ip := range ubcips {
		_, iprange, err := net.ParseCIDR(ip)
		if err != nil {
			log.Panicf("Error parsing ip (%s): %v\n", ip, err)
		}
		ubcranges = append(ubcranges, iprange)
	}
	log.Printf("Loaded %d IP ranges\n", len(ubcips))
}

func IpIsUBC(ipstring string) bool {
	ip := net.ParseIP(ipstring)
	for _, iprange := range ubcranges {
		if iprange.Contains(ip) {
			return true
		}
	}
	return false
}
