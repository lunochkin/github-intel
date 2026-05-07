package main

import (
	"fmt"
	"log"

	"github.com/lunochkin/github-intel/internal/dotenv"
)

func main() {
	if err := dotenv.Load(); err != nil {
		log.Fatalf("load .env: %v", err)
	}
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	fmt.Println("ingestor: placeholder — wire GitHub Archive download + parsing here")
	return nil
}
