package main

import (
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
)

func main() {
	c, err := redis.Dial("tcp", "172.16.18.211:6379")
	if err != nil {
		fmt.Println("Connect to redis error", err)
		return
	}
	defer c.Close()

	x, err := c.Do("AUTH", "authoring")
	if err != nil {
		fmt.Println("redis auth failed:", err)
	}
	fmt.Println("AUTH", x)

	_, err = c.Do("SELECT", 1)
	if err != nil {
		fmt.Println("redis select failed:", err)
	}

	_, err = c.Do("SET", "mykey", "superWang", "EX", "5")
	if err != nil {
		fmt.Println("redis set failed:", err)
	}

	username, err := redis.String(c.Do("GET", "mykey"))
	if err != nil {
		fmt.Println("redis get failed:", err)
	} else {
		fmt.Printf("Get mykey: %v \n", username)
	}

	time.Sleep(8 * time.Second)

	username, err = redis.String(c.Do("GET", "mykey"))
	if err != nil {
		if err == redis.ErrNil {
			fmt.Println("redis get: mykey not exits")
		} else {
			fmt.Println("redis get failed:", err)
		}
	} else {
		fmt.Printf("Get mykey: %v \n", username)
	}
}
