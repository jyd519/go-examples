package main

import (
	"fmt"
	"log"

	"github.com/dop251/goja"
)

// Define a Go struct that we'll expose to JavaScript
type User struct {
	Name  string
	Email string
}

// Method that will be available to JavaScript
func (u *User) GetFullInfo() string {
	return fmt.Sprintf("%s (%s)", u.Name, u.Email)
}

// Global function to be called from JavaScript
func Add(a, b int) int {
	return a + b
}

// Function that accepts a JavaScript callback
func ProcessWithCallback(vm *goja.Runtime, callback goja.Callable) {
	// Create some data to pass to JavaScript
	data := map[string]interface{}{
		"message": "Hello from Go!",
		"code":    200,
	}

	// Convert Go map to JavaScript object
	jsData := vm.ToValue(data)

	// Call the JavaScript callback with our data
	_, err := callback(goja.Undefined(), jsData)
	if err != nil {
		log.Printf("Error in callback: %v", err)
	}
}

func main() {
	// Create a new JavaScript runtime
	vm := goja.New()

	vm.Set(`log`, func(args ...interface{}) {
		fmt.Println(args...)
	})

	// 1. Basic JavaScript evaluation
	val, err := vm.RunString("2 + 2")
	if err != nil {
		log.Fatalf("Failed to run script: %v", err)
	}
	fmt.Printf("2 + 2 = %v\n", val)

	// 2. Working with JavaScript objects
	_, err = vm.RunString(`
		var person = {
			name: "John Doe",
			age: 30,
			greet: function() {
				return "Hello, " + this.name;
			}
		};
	`)
	if err != nil {
		log.Fatalf("Failed to create person object: %v", err)
	}

	// Get JavaScript object in Go
	person := vm.Get("person").ToObject(vm)
	name := person.Get("name").String()
	age := person.Get("age").ToInteger()
	fmt.Printf("Person: %s, Age: %d\n", name, age)

	// Call JavaScript function from Go
	greet, _ := goja.AssertFunction(person.Get("greet"))
	result, _ := greet(goja.Undefined(), person)
	fmt.Printf("Greeting: %s\n", result)

	// 3. Exposing Go objects to JavaScript
	user := &User{Name: "Alice", Email: "alice@example.com"}
	err = vm.Set("user", user)
	if err != nil {
		log.Fatalf("Failed to set user: %v", err)
	}

	// Access Go object methods from JavaScript
	val, err = vm.RunString("user.GetFullInfo()")
	if err != nil {
		log.Fatalf("Failed to call GetFullInfo: %v", err)
	}
	fmt.Printf("User info from JavaScript: %s\n", val)

	// 4. Register global function
	err = vm.Set("add", Add)
	if err != nil {
		log.Fatalf("Failed to register Add function: %v", err)
	}

	// Call global Go function from JavaScript
	val, err = vm.RunString("add(5, 3)")
	if err != nil {
		log.Fatalf("Failed to call Add function: %v", err)
	}
	fmt.Printf("5 + 3 = %v\n", val)

	// 5. Demonstrate callback pattern
	_, err = vm.RunString(`
		// Define a JavaScript callback function
		function handleGoData(data) {
			log("handleGoData:", data.code);
			return data.code;
		}
	`)
	if err != nil {
		log.Fatalf("Failed to define callback: %v", err)
	}

	// Get the JavaScript callback function
	callback, _ := goja.AssertFunction(vm.Get("handleGoData"))

	// Call Go function with JavaScript callback
	ProcessWithCallback(vm, callback)

	// 6. Error handling demonstration
	_, err = vm.RunString("throw new Error('This is a JavaScript error');")
	if err != nil {
		fmt.Printf("Caught JavaScript error: %v\n", err)
	}
}
