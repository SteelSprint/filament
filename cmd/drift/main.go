package main

import (
	"fmt"
	"os"

	driftpin "driftpin"
)

func main() {
	output, code := driftpin.Run(os.Args[1:], ".")
	if output != "" {
		fmt.Println(output)
	}
	os.Exit(code)
}
