package main

import (
	"flag"
	"fmt"

	"github.com/oodaris/autocodex/internal/config"
)

func runConfig(args []string) {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	configPath := fs.String("config", config.ResolveConfigPath(), "Config file path")
	fs.Parse(args)
	fmt.Printf("Config path: %s\n", *configPath)
}
