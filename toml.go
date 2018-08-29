package main

import (
	"fmt"
	"log"

	"github.com/BurntSushi/toml"
)

func main() {
	fmt.Println("Hello, playground")
	d := `
    [users]
    a="foo"
    b="bar"
    c="bar"
    `
	var s struct {
		Users map[string]string
	}

	_, err := toml.Decode(d, &s)
	if err != nil {
		log.Fatal(err)
	}

	// check err!
	fmt.Printf("%#v", s)
}
