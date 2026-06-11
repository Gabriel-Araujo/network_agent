package main

import (
	"fmt"

	"github.com/Gabriel-Araujo/network_agent/pkg/util/config"
)

func main() {
	config := config.Load()

	fmt.Println(config)
}
