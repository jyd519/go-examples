package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {

	fmt.Println("On Unix:")
	root, _ := os.Getwd()
	fmt.Println(root)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}
		if info.IsDir() && path != root {
			// fmt.Printf("skipping a dir without errors: %+v \n", path)
			return filepath.SkipDir
		}
		fmt.Printf("visited file or dir: %q\n", path)
		return nil
	})
	if err != nil {
		fmt.Printf("error walking the path : %v\n", err)
		return
	}
}
