package runner

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"syscall"

	"github.com/go-errors/errors"
	// "github.com/modulesio/isolator/runner/macutil"
	"github.com/modulesio/isolator/runner/policies"
)

var investigateSandbox = os.Getenv("INVESTIGATE_SANDBOX") == "1"

type sandboxExecRunner struct {
	params *RunnerParams
}

var _ Runner = (*sandboxExecRunner)(nil)

func newSandboxExecRunner(params *RunnerParams) (Runner, error) {
	ser := &sandboxExecRunner{
		params: params,
	}
	return ser, nil
}

func (ser *sandboxExecRunner) Prepare() error {
	// make sure we have sandbox-exec
	{
		cmd := exec.Command("sandbox-exec")
		err := cmd.Run()
		if err != nil {
			exitErr := err.(*exec.ExitError)
			waitStatus := exitErr.Sys().(syscall.WaitStatus)
			exitStatus := waitStatus.ExitStatus()
			if (exitStatus != 64) {
				return errors.New("Cannot run sandbox-exec")
		  }
		}
	}

	return nil
}

func (ser *sandboxExecRunner) Run() error {
	params := ser.params

	sandboxProfilePath := filepath.Join(params.InstallFolder, ".isolator", "isolate-app.sb")
	err := os.MkdirAll(filepath.Dir(sandboxProfilePath), 0755)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	sandboxSource := policies.SandboxExecTemplate
	sandboxSource = strings.Replace(
		sandboxSource,
		"{{DIRECTORY_LOCATION}}",
		params.Dir,
		-1, // replace all instances
	)
	sandboxSource = strings.Replace(
		sandboxSource,
		"{{INSTALL_LOCATION}}",
		params.InstallFolder,
		-1, // replace all instances
	)

	err = ioutil.WriteFile(sandboxProfilePath, []byte(sandboxSource), 0644)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	workDir, err := ioutil.TempDir("", "butler-shim-bundle")
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer os.RemoveAll(workDir)

	shimBundlePath := filepath.Join(
		workDir,
		"shim.sh",
	)

	shimBinaryPath := filepath.Join(
		shimBundlePath,
		/* "Contents",
		"MacOS",
		binaryName, */
	)
	err = os.MkdirAll(filepath.Dir(shimBinaryPath), 0755)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	shimBinaryContents := fmt.Sprintf(`#!/bin/bash
		cd "%s"
		sandbox-exec -f "%s" "%s" "$@"
		`,
		params.Dir,
		sandboxProfilePath,
		params.FullTargetPath,
	)

	err = ioutil.WriteFile(shimBinaryPath, []byte(shimBinaryContents), 0755)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	/* err = os.Symlink(
		filepath.Join(realBundlePath, "Contents", "Resources"),
		filepath.Join(shimBundlePath, "Contents", "Resources"),
	)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = os.Symlink(
		filepath.Join(realBundlePath, "Contents", "Info.plist"),
		filepath.Join(shimBundlePath, "Contents", "Info.plist"),
	)
	if err != nil {
		return errors.Wrap(err, 0)
	} */

	if investigateSandbox {
		for {
			time.Sleep(1 * time.Second)
		}
	}

	return RunAppBundle(
		params,
		shimBundlePath,
	)
}
