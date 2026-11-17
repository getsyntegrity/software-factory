// Command aifactory is the entry point for the Go implementation.
package main

import "fmt"

// version is set at build time.
var version = "dev"

func main() {
	fmt.Println("aifactory", version)
}
