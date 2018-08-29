package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
)

func newPool(redisUrl string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 60 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(redisUrl)
			if err != nil {
				log.Printf("[ERROR]: newPool(redisUrl=%q) returned err=%v", redisUrl, err)
				return nil, err
			}
			log.Printf("[DEBUG]: configured to connect to REDIS_URL=%s", redisUrl)
			return c, err
		},
		MaxActive: 1000,
		TestOnBorrow: func(c redis.Conn, t time.Time) error {

			_, err := c.Do("PING")
			if err != nil {
				log.Printf("[ERROR]: TestOnBorrow failed healthcheck to redisUrl=%q err=%v",
					redisUrl, err)
			}
			return err
		},
		Wait: true, // pool.Get() will block waiting for a connection to free
	}
}
func main() {
	url := os.Getenv("REDIS_URL")

	p := newPool(url)
	for {
		// Get a connection from a pool
		c := p.Get()
		psc := redis.PubSubConn{Conn: c}

		// Set up subscriptions
		err := psc.Subscribe("example")
		if err != nil {
			fmt.Printf("ERROR: %#v", err)
			return
		}

		// While not a permanent error on the connection.
		for c.Err() == nil {
			switch v := psc.Receive().(type) {
			case redis.Message:
				fmt.Printf("%s: message: %s\n", v.Channel, v.Data)
			case redis.Subscription:
				fmt.Printf("%s: %s %d\n", v.Channel, v.Kind, v.Count)
			case error:
				fmt.Println(v)
			}
		}
		c.Close()
	}

}
