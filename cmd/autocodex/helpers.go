package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func writeJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func exitErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	os.Exit(1)
}
