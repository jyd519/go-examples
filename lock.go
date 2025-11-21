package main

import (
	"fmt"
	"os"
)

func main() {
	filePath := os.Args[1]

	// Open the file for exclusive creation
	file, err := os.OpenFile(filePath, os.O_EXCL | os.O_RDWR, 0644)
	if err != nil {
		if os.IsExist(err) {
			fmt.Println("File already exists:", err)
		} else {
			fmt.Println("Error opening file:", err)
		}
		return
	}

	fmt.Println("File created successfully.")
	var b byte
	fmt.Scanf("%c", &b)

	defer file.Close()

}
