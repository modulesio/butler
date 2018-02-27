// +build !linux

package linuxsandbox

import (
  "github.com/modulesio/isolator/mansion"
  "github.com/itchio/wharf/state"
)

func Register(ctx *mansion.Context) {
	// don't register anything
	return
}

func Check(consumer *state.Consumer) error {
  return nil
}
