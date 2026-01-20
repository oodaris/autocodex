package state

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func randSuffix(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
