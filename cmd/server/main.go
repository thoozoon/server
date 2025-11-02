package main

import (
	"os"
	"server"
)

func main() {
	if len(os.Args) != 2 {
		panic("Expected config file path as command-line argument.")
	}
	server.Run(os.Args[1])
}
