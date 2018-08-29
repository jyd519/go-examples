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
	p := newPool(os.Getenv("REDIS_URL"))
	c := p.Get()

	defer c.Close()

	for i := 0; i < 5; i++ {
		c.Send("INCR", "maerf0x0")
	}

	c.Send("SET", "username", "maerf0x0")
	c.Send("GET", "username")

	hmsetArgs := redis.Args{"myhashpipe"}             // key
	hmsetArgs = hmsetArgs.Add("username", "maerf0x0") // value pairs
	hmsetArgs = hmsetArgs.Add("usertype", "github")
	c.Send("HMSET", hmsetArgs...)
	c.Send("HGETALL", "myhashpipe")

	c.Send("INCR", "unholy", 939, 3993) // ERROR, too many args

	// Note: this do triggers the actual sending/execution of the pipelined commands
	// afaik no commands will run until this DO triggers them.
	res, err := redis.Values(c.Do(""))

	if err != nil {
		fmt.Println("ERROR ", err)
	}

	// Now we can range over the responses looking for things of interest like redis.Error or
	// redis.ErrNil
	for i, r := range res {
		switch t := r.(type) {
		case redis.Error:
			fmt.Printf("res[%d] is redis.Error %v\n", i, r)
		case string:
			a, _ := redis.String(r, nil)
			fmt.Printf("res[%d] is redis.String %v\n", i, a)
		case int64:
			fmt.Printf("res[%d] is int64=%d \n", i, r)
		case []interface{}:
			// You have to use context to know
			s, _ := redis.Strings(r, nil)
			fmt.Println("  Could be redis.Strings , use context to know what. ", s)
			if len(r.([]interface{}))%2 == 0 { // even number, maybe hashmap
				m, _ := redis.StringMap(r, nil)
				fmt.Println("  Could be map[string]string ", m)
			}
		default:
			fmt.Printf("res[%d] is type=%T %#v  %v\n", i, r, r, t)

		}

	}
}
