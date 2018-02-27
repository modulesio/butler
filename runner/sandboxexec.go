package runner

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/modulesio/butler/runner/macutil"
	"github.com/modulesio/butler/runner/policies"
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
		cmd := exec.Command("sandbox-exec", "-n", "no-network", "true")
		err := cmd.Run()
		if err != nil {
			fmt.Printf("While verifying sandbox-exec: %s", err.Error())
			return errors.New("Cannot set up itch.io sandbox, see logs for details")
		}
	}

	return nil
}

func (ser *sandboxExecRunner) Run() error {
	params := ser.params

	fmt.Printf("Creating shim app bundle to enable sandboxing")
	realBundlePath := params.FullTargetPath

  binaryPath := realBundlePath;
	/* binaryPath, err := macutil.GetExecutablePath(realBundlePath)
	if err != nil {
		return errors.Wrap(err, 0)
	} */
	// binaryName := filepath.Base(binaryPath)

	sandboxProfilePath := filepath.Join(params.Dir, ".isolator", "isolate-app.sb")
	fmt.Printf("Writing sandbox profile to (%s)", sandboxProfilePath)
	err := os.MkdirAll(filepath.Dir(sandboxProfilePath), 0755)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	userLibrary, err := macutil.GetLibraryPath()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	sandboxSource := policies.SandboxExecTemplate
	sandboxSource = strings.Replace(
		sandboxSource,
		"{{USER_LIBRARY}}",
		userLibrary,
		-1, /* replace all instances */
	)
	sandboxSource = strings.Replace(
		sandboxSource,
		"{{INSTALL_LOCATION}}",
		params.InstallFolder,
		-1, /* replace all instances */
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
		filepath.Base(realBundlePath),
	)
	fmt.Printf("Generating shim bundle as (%s)", shimBundlePath)

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
		binaryPath,
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
		fmt.Printf("Wrote shim app to (%s), waiting forever because INVESTIGATE_SANDBOX is set to 1")
		for {
			time.Sleep(1 * time.Second)
		}
	}

	fmt.Printf("All set, hope for the best")

	return RunAppBundle(
		params,
		shimBundlePath,
	)
}
