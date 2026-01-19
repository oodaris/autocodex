package main

import _ "embed"

//go:embed embedded_config.example.yaml
var embeddedConfigExample []byte
