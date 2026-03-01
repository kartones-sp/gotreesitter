//go:build cgo && treesitter_c_parity

package cgoharness

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
