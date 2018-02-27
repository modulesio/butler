package runner

import (
	"fmt"
	// "io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-errors/errors"
  "github.com/itchio/wharf/state"
	"github.com/modulesio/butler/cmd/elevate"
	"github.com/modulesio/butler/cmd/operate"
  "github.com/modulesio/butler/installer"
	// "github.com/modulesio/butler/runner/policies"
	"github.com/modulesio/butler/cmd/linuxsandbox"
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
		fmt.Printf("Sandbox check failed: %s", err.Error())

		/* ctx := wr.params.Ctx
		conn := wr.params.Conn

		var r buse.AllowSandboxSetupResponse
		err := conn.Call(ctx, "AllowSandboxSetup", &buse.AllowSandboxSetupParams{}, &r)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if !r.Allow {
			return operate.ErrAborted
		} */
		fmt.Printf("Proceeding with sandbox setup...")

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

    fmt.Printf("check exit code %v", res.ExitCode)

    if res.ExitCode != 0 {
			if res.ExitCode == elevate.ExitCodeAccessDenied {
				return operate.ErrAborted
			}
		}

		err = installer.CheckExitCode(res.ExitCode, err)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		fmt.Printf("Proceeding with sandbox setup...")
  }

  executable, err := os.Executable()
  if err != nil {
    return errors.Wrap(err, 0)
  }

  firejailPath := filepath.Join(filepath.Dir(executable), "bin", "bwrap")

	/* sandboxProfilePath := filepath.Join(params.Dir, ".isolator", "isolate-app.profile")
	fmt.Printf("Writing sandbox profile to (%s)", sandboxProfilePath)
	err = os.MkdirAll(filepath.Dir(sandboxProfilePath), 0755)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	sandboxSource := policies.FirejailTemplate
	err = ioutil.WriteFile(sandboxProfilePath, []byte(sandboxSource), 0644)
	if err != nil {
		return errors.Wrap(err, 0)
	} */

	fmt.Printf("Running (%s) through firejail", params.FullTargetPath)

	var args []string
	args = append(args, "--ro-bind", "/usr", "/usr", "--ro-bind", "/bin", "/bin", "--ro-bind", "/sbin", "/sbin", "--bind", params.Dir, params.Dir, "--ro-bind", "/lib", "/lib", "--ro-bind", "/lib64", "/lib64", "--proc", "/proc", "--dev", "/dev", "--unshare-all")
	args = append(args, params.FullTargetPath)
	args = append(args, params.Args...)

  fmt.Printf("firejail command %s %v %v %v", firejailPath, args, params.Stdout, params.Stderr)

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
