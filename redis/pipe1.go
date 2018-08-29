package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/gomodule/redigo/redis"
)

func send(conn redis.Conn, n int) {
	for i := 0; i < n; i++ {
		if err := conn.Send("SET", "key", "value3"); err != nil {
			log.Fatal(err)
		}
	}
	if err := conn.Flush(); err != nil {
		log.Fatal(err)
	}
}

func receive(conn redis.Conn, n int) {
	for i := 0; i < n; i++ {
		_, err := conn.Receive()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	totalRequests := flag.Int("n", 100000, "Total number of `requests`")
	flag.Parse()
	conn, err := redis.Dial("tcp", "172.16.18.211:6379")
	if err != nil {
		log.Fatal(err)
	}
	conn.Do("AUTH", "authoring")
	start := time.Now()
	go send(conn, *totalRequests)
	// conn.Do("")
	receive(conn, *totalRequests)
	t := time.Since(start)
	fmt.Printf("%d requests completed in %f seconds\n", *totalRequests, float64(t)/float64(time.Second))
	fmt.Printf("%f requests / second\n", float64(*totalRequests)*float64(time.Second)/float64(time.Since(start)))
}
