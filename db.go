package main

import (
	"crypto/sha1"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/gomodule/redigo/redis"
)

const HOUR int = 60 * 60
const DAY int = HOUR * 24
const WEEK int = DAY * 7

var pool *redis.Pool
var seededRand *rand.Rand

func DBConnect(serverUrl string) {
	pool = &redis.Pool{
		MaxIdle:   80,
		MaxActive: 12000,
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(serverUrl)
			if err != nil {
				log.Panic(err.Error())
			}
			return c, err
		},
	}
	seededRand = rand.New(
		rand.NewSource(time.Now().UnixNano()))
}

func DBPing() (string, error) {
	conn := pool.Get()
	defer conn.Close()

	return redis.String(conn.Do("PING"))
}

func GetStats() ([]string, error) {
	conn := pool.Get()
	defer conn.Close()

	return redis.Strings(conn.Do("KEYS", "r:*"))
}

func AddReport(report Report) error {
	conn := pool.Get()
	defer conn.Close()

	rint := 10000 + seededRand.Intn(89999)
	_, err := conn.Do("SET", fmt.Sprintf("r:%s:%s:%s:%d", report.Code, report.Number, report.Section, rint), "", "EX", WEEK)
	if err != nil {
		log.Printf("Error adding entry: %v\n", err)
	}
	return err
}

func RateLimit(id string) (bool, error) {
	rlid := fmt.Sprintf("rl:%x", sha1.Sum([]byte(fmt.Sprintf("SnickitsMark:%s:%d:%d:%d", id, time.Now().Day(), time.Now().Month(), time.Now().Year()))))

	conn := pool.Get()
	defer conn.Close()

	rlval, err := redis.Int(conn.Do("INCR", rlid))
	if err != nil {
		fmt.Printf("Error incrementing value: %v\n", err)
		return false, err
	}
	_, err = conn.Do("EXPIRE", rlid, DAY)
	if err != nil {
		log.Printf("Error setting expiry: %v\n", err)
	}
	if rlval <= rlmax {
		return true, nil
	} else {
		return false, nil
	}
}

func RateLimitEntry(id string, entry Report) (bool, error) {
	if testmode {
		log.Println("testmode: Skipping RateLimitEntry check")
		return true, nil
	}

	rlid := fmt.Sprintf("rl:%x", sha1.Sum([]byte(fmt.Sprintf("SnickitsMark:%s:%s%s:%d:%d:%d", id, entry.Code, entry.Number, time.Now().Day(), time.Now().Month(), time.Now().Year()))))

	conn := pool.Get()
	defer conn.Close()

	_, err := redis.String(conn.Do("SET", rlid, "", "EX", HOUR, "NX"))
	if err == redis.ErrNil {
		return false, nil
	}
	if err != nil {
		fmt.Printf("Error incrementing value: %v\n", err)
		return false, err
	}
	return true, nil
}
