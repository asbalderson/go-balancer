package main

import (
	"backend/internal/config"
	"fmt"
	"os"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		fmt.Printf("its broked, %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("We loaded the config from main: %+v\n", cfg)
}
