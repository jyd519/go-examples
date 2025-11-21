package main
// add TDM-GCC\bin to PATH

import (
  "github.com/rosbit/go-quickjs"
  "fmt"
)
// function to be called by Javascript
func adder(a1 float64, a2 float64) float64 {
    return a1 + a2
}
func main() {
  ctx, err := quickjs.NewContext()
  if err != nil {
    fmt.Printf("%v\n", err)
    return
  }

  res, _ := ctx.Eval("a + b", map[string]interface{}{
     "a": 10,
     "b": 1,
  })
  fmt.Println("result is:", res)


  res, _ = ctx.Eval("adder(a, b)", map[string]interface{}{
		"adder": adder,
     "a": 10,
     "b": 100,
  })

  fmt.Println("result is:", res)
}
