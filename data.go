package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"sort"
	"strings"
)

var CoursesMap map[string]string

var (
	reNumber  *regexp.Regexp
	reSection *regexp.Regexp
)

type pair struct {
	Key   string
	Value int
}
type pairList []pair

func (p pairList) Len() int           { return len(p) }
func (p pairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p pairList) Less(i, j int) bool { return p[i].Value > p[j].Value }

type Course struct {
	Rank        int
	SubjectCode string
	Number      string
	Section     string
	Name        string
	Reports     int
}

func (c Course) IsHousing() bool { return c.SubjectCode[:2] == "0R" }

type Summary struct {
	Error   bool
	Courses []Course
}
type Report struct {
	Code    string
	Number  string
	Section string
}
type SubmitPage struct {
	Error    bool
	ErrorMsg string
	GoBack   bool
}

func (r Report) IsValid() bool {
	if len(r.Code) > 4 || len(r.Number) > 3 || len(r.Section) > 3 {
		return false
	}
	if _, ok := CoursesMap[r.Code]; !ok {
		return false
	}
	if r.Code[:2] == "0R" {
		// Residence
		r.Number = ""
		r.Section = ""
	} else {
		// Course
		if !reNumber.MatchString(r.Number) || reSection.MatchString(r.Section) {
			return false
		}
	}
	return true
}

func LoadCourses() {
	coursedata, err := ioutil.ReadFile("courses.json")
	if err != nil {
		log.Panicf("Error reading courses file: %v\n", err)
	}

	err = json.Unmarshal(coursedata, &CoursesMap)
	if err != nil {
		log.Panicf("Error parsing courses file: %v\n", err)
	}
	// Load Regex too
	reNumber = regexp.MustCompile(`^[1-5][0-9]{2}$`)
	reSection = regexp.MustCompile(`^[L0-9][0-9][0-9A-Z]$`)
}

func getCourse(rank int, code string, number string, section string, reports int) Course {
	return Course{
		Rank:        rank,
		SubjectCode: code,
		Number:      number,
		Section:     section,
		Name:        CoursesMap[code],
		Reports:     reports,
	}
}

func GetSummary() Summary {
	// Fetch from DB
	dbdata, err := GetStats()
	if err != nil {
		log.Printf("DB Error loading summary: %v\n", err)
		return Summary{Courses: []Course{}, Error: true}
	}
	// Count Reports
	var reports map[string]int = make(map[string]int)
	for _, dbentry := range dbdata {
		data := ""
		if anon {
			data = dbentry[2 : len(dbentry)-6]
			details := strings.SplitN(data, ":", 3)
			data = fmt.Sprintf("%s:%s:", details[0], details[1])
		} else {
			data = dbentry[2 : len(dbentry)-6]
		}
		reports[data]++
	}
	// Sort Courses
	r := make(pairList, len(reports))
	i := 0
	for k, v := range reports {
		r[i] = pair{k, v}
		i++
	}
	sort.Sort(r)
	// List Courses
	courses := []Course{}
	rank := 0
	lastval := -1
	count := 0
	for _, course := range r {
		if count > maxsummary {
			break
		}
		if course.Value >= minreports {
			cdata := strings.SplitN(course.Key, ":", 3)
			if lastval != course.Value {
				rank++
			}
			courses = append(courses, getCourse(rank, cdata[0], cdata[1], cdata[2], course.Value))
			lastval = course.Value
		} else {
			break
		}
		count++
	}
	log.Println("Generated courses summary")
	return Summary{Courses: courses, Error: false}
}
