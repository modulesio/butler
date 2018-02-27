package runner

import (
	"fmt"
	"regexp"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"runtime"

	"github.com/go-errors/errors"
	"github.com/modulesio/butler/mansion"
	"github.com/modulesio/butler/manager"
	"github.com/modulesio/butler/runner"
)

var args = struct {
	directory     *string
  installPath     *string
  // prereqsPath     *string
	command *[]string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("runner", "Runs a command").Default()
	args.directory = cmd.Flag("directory", "The working directory for the command").String()
  args.installPath = cmd.Flag("installPath", "Temporary install path for sandboxing").String()
  // args.prereqsPath = cmd.Flag("prereqsPath", "Prerequisites path for sandbox tools").Hidden().String()
	args.command = cmd.Arg("command", "A command to run, with arguments").Strings()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx))
}

func Do(ctx *mansion.Context) error {
	command := *args.command

  var matched bool
  if (len(command) > 0) {
    r, err := regexp.Compile("^(?:/|\\.|[a-zA-Z]:\\\\)")
    if err != nil {
      return errors.Wrap(err, 0)
    }

    matched = r.MatchString(command[0])
  } else {
    matched = false
  }
  if (!matched) {
    var args []string;
    ctx.App.Usage(args);
    return nil;
  }

	var directory string
  if (*args.directory != "") {
    directory = *args.directory
  } else {
    directory = filepath.Dir(command[0])
  }
  var installPath string
  if (*args.installPath != "") {
    installPath = *args.installPath
  } else {
    installPath = directory
  }
  /* var prereqsPath string
  if (*args.prereqsPath != "") {
    prereqsPath = *args.prereqsPath
  } else {
    prereqsPath = directory
  } */

  fmt.Printf("running %s %s %d", command[0], directory, *args.directory != "")

  runParams := &runner.RunnerParams{
		// Consumer: consumer,
		// Conn:     conn,
		// Ctx:      ctx,

		Sandbox: true,

		FullTargetPath: command[0],

		Name:   directory,
		Dir:    directory,
		Args:   command[1:],
		Env:    os.Environ(),
		Stdin: os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,

		// PrereqsDir:    prereqsPath,
		// Credentials:   params.Credentials,
		InstallFolder: installPath,
		Runtime:       manager.CurrentRuntime(),
	}

  run, err := runner.GetRunner(runParams)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = run.Prepare()
	if err != nil {
		return errors.Wrap(err, 0)
	}

  exitCode, err := interpretRunError(run.Run())
  if err != nil {
    return errors.Wrap(err, 0)
  }

  if exitCode != 0 {
    var signedExitCode = int64(exitCode)
    if runtime.GOOS == "windows" {
      // Windows uses 32-bit unsigned integers as exit codes, although the
      // command interpreter treats them as signed. If a process fails
      // initialization, a Windows system error code may be returned.
      signedExitCode = int64(int32(signedExitCode))

      // The line above turns `4294967295` into -1
    }

    exeName := filepath.Base(runParams.FullTargetPath)
    msg := fmt.Sprintf("Exit code 0x%x (%d) for (%s)", uint32(exitCode), signedExitCode, exeName)
    fmt.Printf(msg)

    /* if runDuration.Seconds() > 10 {
      fmt.Printf("That's after running for %s, ignoring non-zero exit code", runDuration)
    } else { */
      return errors.New(msg)
    // }
  }

  /* launcherParams := &LauncherParams{
		// Conn:     conn,
		Ctx:      ctx,
		// Consumer: consumer,

		FullTargetPath: "/tmp",
		// Candidate:      candidate,
		// AppManifest:    appManifest,
		// Action:         manifestAction,
		Sandbox:        true,
		Args:           [],
		Env:            env,

		PrereqsDir:    "/tmp/prereqs",
		// Credentials:   params.Credentials,
		InstallFolder: "/tmp/install",
		Runtime:       manager.CurrentRuntime(),
	}

	err = launcher.Do(launcherParams)
	if err != nil {
		return errors.Wrap(err, 0)
	} */

	/* cmd := exec.Command(command[0], command[1:]...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		return errors.Wrap(err, 0)
	} */

	return nil
}

func interpretRunError(err error) (int, error) {
	if err != nil {
		if exitError, ok := AsExitError(err); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus(), nil
			}
		}

		return 127, err
	}

	return 0, nil
}

func AsExitError(err error) (*exec.ExitError, bool) {
	if err == nil {
		return nil, false
	}

	if se, ok := err.(*errors.Error); ok {
		return AsExitError(se.Err)
	}

	if ee, ok := err.(*exec.ExitError); ok {
		return ee, true
	}

	return nil, false
}
