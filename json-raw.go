package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type Message struct {
	Event string
	Data  json.RawMessage // how data is parsed depends on the event
}

type CreateMessage struct {
	ID int `json:"id"`
}

var evt = []byte(`{"event": "create", "data" :{"id":5 }}`)

func main() {
	var message Message
	log.Println("json ", string(evt))
	json.Unmarshal(evt, &message)

	fmt.Printf("message: %+v\n", message)
	log.Println("message.Event", message.Event)
	log.Println("message.Data", string(message.Data))

	var message2 = new(CreateMessage)
	err := json.Unmarshal(message.Data, &message2)
	log.Printf("message2: %+v\n", message2)
	log.Println(err)
}
