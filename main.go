package main

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/umee-network/umeed-indexer/cli"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("\n err loading env %s", err.Error())
	}

	// calls cmd to execute.
	if err := cli.Execute(); err != nil {
		fmt.Printf("\n err running command line %s", err.Error())
		return
	}
}
