// Command "validate": reads a candidate config from stdin (or a file) and
// reports whether it's syntactically valid FRR configuration.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	path := flag.String("file", "", "config file to validate (default: stdin)")
	protocol := flag.String("protocol", "", "protocol hint (bgp, ospf, isis...) -- wraps a bare snippet in its required context if needed")
	flag.Parse()

	var data []byte
	var err error
	if *path != "" {
		data, err = os.ReadFile(*path)
	} else {
		data, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "read input:", err)
		os.Exit(1)
	}

	config := string(data)
	if *protocol != "" {
		config = WrapSnippet(*protocol, config)
	}

	result, err := ValidateConfig(context.Background(), config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "validation error:", err)
		os.Exit(1)
	}

	if result.Valid {
		fmt.Println("VALID")
		return
	}
	fmt.Println("INVALID")
	for _, e := range result.Errors {
		fmt.Printf("  line %d: %s\n", e.Line, e.Message)
	}
	os.Exit(1)
}
