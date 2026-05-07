package main

import (
	"fmt"
	"log"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	fmt.Println("ingestor: placeholder — wire GitHub Archive download + parsing here")
	return nil
}
