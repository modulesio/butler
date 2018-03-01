package runner

import (
	"os"
	"os/exec"

	"github.com/go-errors/errors"
	// "github.com/modulesio/isolator/runner/macutil"
)

type appRunner struct {
	params *RunnerParams
}

var _ Runner = (*appRunner)(nil)

func newAppRunner(params *RunnerParams) (Runner, error) {
	ar := &appRunner{
		params: params,
	}
	return ar, nil
}

func (ar *appRunner) Prepare() error {
	// nothing to prepare
	return nil
}

func (ar *appRunner) Run() error {
	params := ar.params

	return RunAppBundle(
		params,
		params.FullTargetPath,
	)
}

func RunAppBundle(params *RunnerParams, binPath string) error {
	cmd := exec.Command(binPath, params.Args...)
	cmd.Stdin = os.Stdin
  cmd.Stdout = os.Stdout
  cmd.Stderr = os.Stderr
	// I doubt this matters
	cmd.Dir = params.Dir
	cmd.Env = params.Env
	// 'open' does not relay stdout or stderr, so we don't
	// even bother setting them

	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
