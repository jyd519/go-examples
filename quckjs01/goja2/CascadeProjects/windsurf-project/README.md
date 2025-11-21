# Goja Examples

This project demonstrates the basic usage of the Goja package, showcasing how to export Golang functions and structs to JavaScript.

## Features

- Exporting simple functions from Go to JavaScript
- Exporting structs and methods to JavaScript
- Running JavaScript code within a Go runtime

## Prerequisites

- Go 1.21 or later
- Goja package

## Running the Example

```bash
go mod tidy
go run src/main.go
```

## Explanation

The example demonstrates two key scenarios:
1. Exporting a simple function `Add` that can be called from JavaScript
2. Exporting a `Person` struct with a method, showing more complex interoperability

The JavaScript code shows how these exported functions and structs can be used directly in the JavaScript runtime.
