package main

import (
	"log"

	"github.com/kilianp07/v2g/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
