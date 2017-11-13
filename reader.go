package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

func testWriteTo() {
	reader := bytes.NewReader([]byte("Go语言中文网\n"))
	reader.WriteTo(os.Stdout)

	io.Copy(os.Stdout, strings.NewReader("Go语言中文网\n"))
}

func testSeek() {
	reader := strings.NewReader("Go语言中文网")
	reader.Seek(-6, os.SEEK_END)
	r, _, _ := reader.ReadRune()
	fmt.Printf("%c\n", r)
}

func main() {
	testWriteTo()
	testSeek()
	reader := strings.NewReader("Go语言中文网")
	p := make([]byte, 6)
	n, err := reader.ReadAt(p, 2)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s, %d\n", p, n) //语言, 6
}
