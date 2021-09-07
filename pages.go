package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
)

var (
	home   *template.Template
	report *template.Template
	submit *template.Template
	terms  []byte
)

func loadPage(name string) *template.Template {
	tmpl, err := template.ParseFiles(name)

	if err != nil {
		log.Panicf("Error loading '%s' page: %v\n", name, err)
	}

	return tmpl
}

func LoadPages() {
	home = loadPage("www/home.html")
	report = loadPage("www/report.html")
	submit = loadPage("www/submit.html")
	var err error
	terms, err = ioutil.ReadFile("www/terms.html")
	if err != nil {
		log.Panicf("Error loading terms page: %v\n", err)
	}
}

func WriteHome(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		w.Header().Add("Cache-Control", "max-age=600") // 10m
		home.Execute(w, GetSummary())
	} else {
		Write404(w, r)
	}
}

func WriteReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Cache-Control", "max-age=86400") // 24h
	report.Execute(w, CoursesMap)
}

func WriteTerms(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Cache-Control", "max-age=86400") // 24h
	w.Header().Add("Content-Type", "text/html; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	w.Write(terms)
}

func WriteStyle(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Cache-Control", "max-age=600") // 10m
	http.ServeFile(w, r, "www/style.css")
}

func Write404(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Cache-Control", "max-age=3600") // 60m
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Not Found"))
}

func WriteSubmit(w http.ResponseWriter, r *http.Request) {
	// Check IP range
	ip, err := getIP(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		submit.Execute(w, SubmitPage{
			Error:    true,
			ErrorMsg: "It's not you! Something went wrong, try again later.",
			GoBack:   true,
		})
		return
	}
	if !IpIsUBC(ip) {
		w.WriteHeader(http.StatusTooManyRequests)
		submit.Execute(w, SubmitPage{
			Error:    true,
			ErrorMsg: "You must be connected to UBC secure or use the UBC VPN.",
			GoBack:   false,
		})
		return
	}
	// Check form data
	err = r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		submit.Execute(w, SubmitPage{
			Error:    true,
			ErrorMsg: "There was an error with your entry, try again",
			GoBack:   true,
		})
		return
	}
	rp := Report{
		Code:    r.PostForm.Get("course"),
		Number:  r.PostForm.Get("number"),
		Section: r.PostForm.Get("section"),
	}
	if !rp.IsValid() {
		w.WriteHeader(http.StatusBadRequest)
		submit.Execute(w, SubmitPage{
			Error:    true,
			ErrorMsg: "There was an error with your entry.",
			GoBack:   true,
		})
		return
	}
	// Check Ratelimit (IP)
	ok, err := RateLimit(ip)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		submit.Execute(w, SubmitPage{
			Error:    true,
			ErrorMsg: "It's not you! Something went wrong, try again later.",
			GoBack:   true,
		})
		return
	}
	if !ok {
		w.WriteHeader(http.StatusTooManyRequests)
		submit.Execute(w, SubmitPage{
			Error:    true,
			ErrorMsg: "Too many reports for today",
			GoBack:   false,
		})
		return
	}
	// Check Ratelimit (IP+Report)
	ok, err = RateLimitEntry(ip, rp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		submit.Execute(w, SubmitPage{
			Error:    true,
			ErrorMsg: "It's not you! Something went wrong, try again later.",
			GoBack:   true,
		})
		return
	}
	if !ok {
		w.WriteHeader(http.StatusTooManyRequests)
		submit.Execute(w, SubmitPage{
			Error:    true,
			ErrorMsg: "You can't report the same entry twice a day",
			GoBack:   false,
		})
		return
	}
	// Add entry to DB
	err = AddReport(rp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		submit.Execute(w, SubmitPage{
			Error:    true,
			ErrorMsg: "It's not you! Something went wrong, try again later.",
			GoBack:   true,
		})
		return
	}
	// Success
	submit.Execute(w, SubmitPage{
		Error:    false,
		ErrorMsg: "",
		GoBack:   true,
	})
}

func getIP(r *http.Request) (string, error) {
	ip := r.Header.Get("X-Forwarded-For")
	netIP := net.ParseIP(ip)
	if netIP != nil {
		return ip, nil
	}
	ips := r.Header.Get("CF-Connecting-IP")
	splitIps := strings.Split(ips, ",")
	for _, ip := range splitIps {
		netIP := net.ParseIP(ip)
		if netIP != nil {
			return ip, nil
		}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}
	netIP = net.ParseIP(ip)
	if netIP != nil {
		log.Println("Warning: using direct IP")
		return ip, nil
	}
	return "", fmt.Errorf("no valid ip found")
}
