package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func main() {
	// 	const jsonStream = `
	// 	[
	// 		{"Name": "Ed", "Text": "Knock knock."},
	// 		{"Name": "Sam", "Text": "Who's there?"},
	// 		{"Name": "Ed", "Text": "Go fmt."},
	// 		{"Name": "Sam", "Text": "Go fmt who?"},
	// 		{"Name": "Ed", "Text": "Go fmt yourself!"}
	// 	]
	// `

	fmt.Println("Write json")
	{
		f, err := os.Create("./test.json")
		if err != nil {
			fmt.Print(err)
			return
		}
		defer f.Close()
		w := bufio.NewWriter(f)
		w.WriteString("[")
		for i := 0; i < 5000000; i++ {
			w.WriteString(fmt.Sprintf(`{"Name": "%d", "Text": "Go fmt. %d"},`, i, i))
		}
		w.WriteString(fmt.Sprintf(`{"Name": "test", "Text": "Go fmt. "}`))
		w.WriteString("]")
		w.Flush()
	}

	type Message struct {
		Name, Text string
	}

	fmt.Println("Parse json")
	f, err := os.Open("./test.json")
	if err != nil {
		fmt.Print(err)
		return
	}

	dec := json.NewDecoder(bufio.NewReader(f))
	// dec := json.NewDecoder(strings.NewReader(jsonStream))

	// read open bracket
	t, err := dec.Token()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%T: %v\n", t, t)

	// while the array contains values
	for dec.More() {
		var m Message
		// decode an array value (Message)
		err := dec.Decode(&m)
		if err != nil {
			log.Fatal(err)
		}

		if m.Name == "100000" {
			fmt.Printf("%v: %v\n", m.Name, m.Text)
		}
	}

	// read closing bracket
	t, err = dec.Token()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%T: %v\n", t, t)

}
