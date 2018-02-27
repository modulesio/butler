package runner

import (
	"fmt"
	"io"
	"runtime"

	// "github.com/modulesio/isolator/buse"
	// "github.com/modulesio/isolator/cmd/operate"
	"github.com/modulesio/isolator/manager"
	// "github.com/itchio/wharf/state"
  // "github.com/modulesio/isolator/mansion"
)

type RunnerParams struct {
	// Consumer *state.Consumer
	// Conn     operate.Conn
	// Ctx      context.Context

	Sandbox bool

	FullTargetPath string

	Name   string
	Dir    string
	Args   []string
	Env    []string
	Stdin io.Reader
	Stdout io.Writer
	Stderr io.Writer

	// PrereqsDir    string
	// Credentials   *buse.GameCredentials
	InstallFolder string
	Runtime       *manager.Runtime
}

type Runner interface {
	Prepare() error
	Run() error
}

func GetRunner(params *RunnerParams) (Runner, error) {
	switch runtime.GOOS {
	case "windows":
		if params.Sandbox {
			return newWinSandboxRunner(params)
		}
		return newSimpleRunner(params)
	case "linux":
		if params.Sandbox {
			return newFirejailRunner(params)
		}
		return newSimpleRunner(params)
	case "darwin":
		if params.Sandbox {
			return newSandboxExecRunner(params)
		}
		return newAppRunner(params)
	}

	return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
}
