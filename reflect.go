package main

import (
	"fmt"
	"reflect"
)

func main() {
	fn := func(p1 string, p2 *string) error {
		fmt.Printf("p1 = %s, p2=%s\n", p1, *p2)
		return fmt.Errorf("ERROR hello")
	}

	t := reflect.TypeOf(fn)
	t1 := t.In(0)
	t2 := t.In(1)
	v := reflect.ValueOf(fn)
	v1 := reflect.New(t1)
	v2 := reflect.New(t2.Elem())
	// v22 := reflect.New(t2.Elem())
	v1.Elem().SetString("abc")
	v2.Elem().SetString("xxxx")
	// fmt.Printf("%+v, %+v\n", v1.Elem(), v22.Elem())
	// v2.Elem().Set(v22)
	args := []reflect.Value{v1.Elem(), v2}
	r := v.Call(args)

	fmt.Printf("%d , %+v\n", len(r), r[0])
	fmt.Printf("%+v\n", r[0].Interface())

	//
	// fmt.Printf("v.Name = %s\n", v.Name())
	// fmt.Printf("v.kind= %v\n", v.Kind())
	// fmt.Printf("v.NumIn= %v\n", v.NumIn())
	// for i := 0; i < v.NumIn(); i++ {
	// 	t := v.In(i)
	// 	fmt.Printf("\t%d = %v %v\n", i, t, t.Kind())
	// 	if t.Kind() == reflect.Ptr {
	// 		t = t.Elem()
	// 	}
	// 	iv := reflect.New(t).Elem()
	// 	iv.SetString("abc")
	// 	fmt.Printf("\t\t %+v\n", iv.Interface())
	// }
	// fmt.Printf("v.NumOut= %v\n", v.NumOut())
	// for i := 0; i < v.NumOut(); i++ {
	// 	t := v.Out(i)
	// 	fmt.Printf("\t%d = %v\n", i, t)
	// 	iv := reflect.New(t).Interface()
	// 	fmt.Printf("\t\t %+v\n", iv)
	// }
}
