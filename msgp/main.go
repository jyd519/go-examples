package main

import (
	"fmt"
)

//go:generate msgp

type Foo struct {
	Bar string  `msg:"bar"`
	Baz float64 `msg:"baz"`
}

func main() {
	fmt.Println("Nothing to see here yet!")
}
