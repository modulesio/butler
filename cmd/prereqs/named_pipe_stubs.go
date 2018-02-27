// +build !windows

package prereqs

import (
	"github.com/modulesio/isolator/buse"
	"github.com/modulesio/isolator/comm"
	"github.com/itchio/wharf/state"
)

type NamedPipe struct {
}

func NewNamedPipe(pipePath string) (*NamedPipe, error) {
	np := &NamedPipe{}

	return np, nil
}

func (np NamedPipe) Consumer() *state.Consumer {
	return comm.NewStateConsumer()
}

func (np NamedPipe) WriteState(taskName string, status buse.PrereqStatus) error {
	msg := PrereqState{
		Type:   "state",
		Name:   taskName,
		Status: status,
	}
	comm.Result(&msg)

	return nil
}
