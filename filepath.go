package main

import (
	"fmt"
	"path/filepath"
)

func main() {
	fmt.Printf("No dots: %q\n", filepath.Ext("index"))         // ''
	fmt.Printf("One dot: %q\n", filepath.Ext("index.js"))      //.js
	fmt.Printf("Two dots: %q\n", filepath.Ext("main.test.js")) //.js

	fmt.Println(filepath.FromSlash("c/sd\\fasd/sdfasdf/sdfsdf"))
	fmt.Println(filepath.ToSlash("c/sdf\\asd/sdfasdf/sdfsdf"))

}
