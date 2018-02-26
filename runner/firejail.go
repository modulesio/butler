package runner

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/runner/policies"
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

	firejailName := fmt.Sprintf("firejail-%s", params.Runtime.Arch())
	firejailPath := filepath.Join(params.PrereqsDir, firejailName, "firejail")

	sandboxProfilePath := filepath.Join(params.InstallFolder, ".itch", "isolate-app.profile")
	fmt.Printf("Writing sandbox profile to (%s)", sandboxProfilePath)
	err := os.MkdirAll(filepath.Dir(sandboxProfilePath), 0755)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	sandboxSource := policies.FirejailTemplate
	err = ioutil.WriteFile(sandboxProfilePath, []byte(sandboxSource), 0644)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	fmt.Printf("Running (%s) through firejail", params.FullTargetPath)

	var args []string
	args = append(args, fmt.Sprintf("--profile=%s", sandboxProfilePath))
	args = append(args, "--")
	args = append(args, params.FullTargetPath)
	args = append(args, params.Args...)

  fmt.Printf("firejail command %s %v", firejailPath, args)

	cmd := exec.Command(firejailPath, args...)
	cmd.Dir = params.Dir
	cmd.Env = params.Env
	cmd.Stdout = params.Stdout
	cmd.Stderr = params.Stderr

	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
