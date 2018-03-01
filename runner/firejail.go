package runner

import (
	// "io/ioutil"
	"os"
	"os/exec"
  "os/user"
	"path/filepath"

	"github.com/go-errors/errors"
  "github.com/itchio/wharf/state"
	"github.com/modulesio/isolator/cmd/elevate"
	"github.com/modulesio/isolator/cmd/operate"
  "github.com/modulesio/isolator/installer"
	// "github.com/modulesio/isolator/runner/policies"
	"github.com/modulesio/isolator/cmd/linuxsandbox"
)

type firejailRunner struct {
	params *RunnerParams
}

var _ Runner = (*firejailRunner)(nil)

func newFirejailRunner(params *RunnerParams) (Runner, error) {
	fr := &firejailRunner{
		params: params,
	}
	return fr, nil
}

func (fr *firejailRunner) Prepare() error {
	// nothing to prepare
	return nil
}

func (fr *firejailRunner) Run() error {
	params := fr.params

  nullConsumer := &state.Consumer{}
	err := linuxsandbox.Check(nullConsumer)
	if err != nil {
		res, err := installer.RunSelf(&installer.RunSelfParams{
			Consumer: nullConsumer,
			Args: []string{
				"--elevate",
				"linuxsandbox",
				"setup",
			},
		})
		if err != nil {
			return errors.Wrap(err, 0)
		}

    if res.ExitCode != 0 {
			if res.ExitCode == elevate.ExitCodeAccessDenied {
				return operate.ErrAborted
			}
		}

		err = installer.CheckExitCode(res.ExitCode, err)
		if err != nil {
			return errors.Wrap(err, 0)
		}
  }

  executable, err := os.Executable()
  if err != nil {
    return errors.Wrap(err, 0)
  }

  firejailPath := filepath.Join(filepath.Dir(executable), "bin", "bwrap")

  usr, err := user.Current()
  if err != nil {
    return errors.Wrap(err, 0)
  }
  configPath := filepath.Join(usr.HomeDir, ".config")
  nvmPath := filepath.Join(usr.HomeDir, ".nvm")

	var args []string
	args = append(args, "--ro-bind", "/usr", "/usr", "--ro-bind", "/bin", "/bin", "--ro-bind", "/sbin", "/sbin", "--bind", params.Dir, params.Dir, "--bind", params.InstallFolder, params.InstallFolder, "--ro-bind", "/lib", "/lib", "--ro-bind", "/lib64", "/lib64", "--ro-bind", "/etc", "/etc", "--ro-bind", configPath, configPath, "--ro-bind", nvmPath, nvmPath, "--proc", "/proc", "--dev", "/dev", "--unshare-all", "--share-net")
	args = append(args, params.FullTargetPath)
	args = append(args, params.Args...)

	cmd := exec.Command(firejailPath, args...)
	cmd.Dir = params.Dir
	cmd.Env = params.Env
	cmd.Stdin = params.Stdin
	cmd.Stdout = params.Stdout
	cmd.Stderr = params.Stderr

	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
