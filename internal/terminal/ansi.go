package terminal

import (
	"bytes"
	"os"
)

var dsrQuery = []byte{0x1b, '[', '6', 'n'}
var dsrResponse = []byte{0x1b, '[', '1', ';', '1', 'R'}

// DSRFilter removes device status report (DSR) queries from PTY output and
// responds with a fixed cursor position. This prevents interactive tools from
// timing out when no terminal emulator is present.
type DSRFilter struct {
	pending []byte
}

func (f *DSRFilter) Filter(pty *os.File, input []byte) []byte {
	if len(input) == 0 {
		return nil
	}
	data := append(f.pending, input...)
	f.pending = nil

	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); {
		if data[i] == 0x1b {
			remaining := data[i:]
			if len(remaining) < len(dsrResponse) {
				if len(remaining) >= len(dsrQuery) && bytes.HasPrefix(remaining, dsrQuery) {
					if pty != nil {
						_, _ = pty.Write(dsrResponse)
					}
					i += len(dsrQuery)
					continue
				}
				if bytes.HasPrefix(dsrQuery, remaining) || bytes.HasPrefix(dsrResponse, remaining) {
					f.pending = append([]byte{}, remaining...)
					break
				}
			} else if bytes.HasPrefix(remaining, dsrQuery) {
				if pty != nil {
					_, _ = pty.Write(dsrResponse)
				}
				i += len(dsrQuery)
				continue
			} else if bytes.HasPrefix(remaining, dsrResponse) {
				i += len(dsrResponse)
				continue
			}
		}
		out = append(out, data[i])
		i++
	}
	return out
}
