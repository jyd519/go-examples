package main

import (
	"fmt"
	"log"

	"github.com/dop251/goja"
)

// Example 1: Simple function export
func Add(a, b int) int {
	return a + b
}

// Example 2: More complex function with struct
type Person struct {
	Name string
	Age  int
}

func CreatePerson(name string, age int) *Person {
	return &Person{Name: name, Age: age}
}

func (p *Person) Introduce() string {
	return fmt.Sprintf("Hi, I'm %s and I'm %d years old", p.Name, p.Age)
}

func main() {
	// Create a new Goja runtime
	vm := goja.New()

	vm.Set("log", log.Println)

	// Example 1: Exporting a simple function
	vm.Set("goAdd", Add)

	// Example 2: Exporting a struct and its method
	vm.Set("createPerson", CreatePerson)

	vm.Set("console", struct {
		Log func(v ...interface{})
	}{
		Log: log.Println,
	})

	// JavaScript code to demonstrate function exports
	jsCode := `
		// Using the exported Go function
		let sum = goAdd(5, 7);
		console.Log('5 + 7 =', sum);
		// Using the exported struct and method
		let person = createPerson('Alice', 30);
		log(person.Introduce());
	`

	// Run the JavaScript code
	_, err := vm.RunString(jsCode)
	if err != nil {
		log.Fatal(err)
	}

	// Example 3: Accessing a Go map from JavaScript
	vm.Set("goMap", map[string]int{"one": 1, "two": 2, "three": 3})

	jsCode = `
		// Accessing the values in the exported Go map
		for (let key in goMap) {
			log("key: ", key, "value: ", goMap[key]);
		}
	`

	_, err = vm.RunString(jsCode)
	if err != nil {
		log.Fatal(err)
	}
}
