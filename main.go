package main

import (
	"flag"
	"log"
	"net/http"
)

var (
	addr       string
	rlmax      int
	anon       bool
	minreports int
	maxsummary int
	testmode   bool
)

func init() {
	var dburl string
	// Parse Flags
	flag.StringVar(&addr, "addr", ":8081", "Server address to listen to")
	flag.StringVar(&dburl, "db", "redis://127.0.0.1:6379/0", "URL for Redis DB")
	flag.IntVar(&rlmax, "rl", 10, "Reports per day per IP")
	flag.BoolVar(&anon, "anon", true, "Increase anonymity of summary")
	flag.IntVar(&minreports, "minreports", 5, "Minimum reports to be added to the summary")
	flag.IntVar(&maxsummary, "maxsummary", 20, "Maximum summary entries")
	flag.BoolVar(&testmode, "testmode", false, "Disable certain safety features")
	flag.Parse()
	// Load Required
	LoadCourses()
	LoadPages()
	ParseIPs()
	// Start DB Pool
	DBConnect(dburl)
	// Test DB
	log.Println("Testing DB Ping...")
	pong, err := DBPing()
	if err != nil {
		log.Panicf("Ping panic!\n%v\n", err)
	}
	log.Printf("Ping response: %v\n", pong)
	if testmode {
		log.Println("WARNING: Test Mode ON, some safety features will be disabled.")
	}
}

func main() {
	// Setup Handles
	log.Println("Setting up server...")
	http.HandleFunc("/", WriteHome)
	http.HandleFunc("/report/", WriteReport)
	http.HandleFunc("/submit/", WriteSubmit)
	http.HandleFunc("/terms/", WriteTerms)
	http.HandleFunc("/style.css", WriteStyle)
	// Run Server
	log.Printf("Starting server (listening on '%v')\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
