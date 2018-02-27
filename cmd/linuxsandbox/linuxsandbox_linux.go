// +build linux

// This package implements a sandbox for Windows. It works by
// creating a less-privileged user, `itch-player-XXXXX`, which
// we hide from login and share a game's folder before we launch
// it (then unshare it immediately after).
//
// If you want to see/manage the user the sandbox creates,
// you can use "lusrmgr.msc" on Windows (works in Win+R)
package linuxsandbox

import (
	"os"
	"fmt"
	"syscall"
	"path/filepath"
	// "time"

	// "github.com/modulesio/isolator/runner/syscallex"

	"github.com/go-errors/errors"
	"github.com/modulesio/isolator/comm"
	"github.com/itchio/wharf/state"

	"github.com/modulesio/isolator/mansion"
)

func Register(ctx *mansion.Context) {
	parentCmd := ctx.App.Command("linuxsandbox", "Use or manage the itch.io sandbox for linux").Hidden()

	{
		cmd := parentCmd.Command("check", "Verify that the sandbox is properly set up").Hidden()
		ctx.Register(cmd, doCheck)
	}

	{
		cmd := parentCmd.Command("setup", "Set up the sandbox (requires elevation)").Hidden()
		ctx.Register(cmd, doSetup)
	}
}

func doCheck(ctx *mansion.Context) {
	ctx.Must(Check(comm.NewStateConsumer()))
}

func Check(consumer *state.Consumer) error {
  executable, err := os.Executable()
  if err != nil {
    return errors.Wrap(err, 0)
  }

  firejailPath := filepath.Join(filepath.Dir(executable), "bin", "bwrap")
  stats, err := os.Lstat(firejailPath)
  if err != nil {
    return errors.Wrap(err, 0)
  }

  isRoot := stats.Sys().(*syscall.Stat_t).Uid == 0
  isSetuid := (stats.Mode() & os.ModeSetuid) != 0

  fmt.Printf("linux sandbox check %v %v", stats.Mode(), isSetuid)

  if (!isRoot || !isSetuid) {
		return errors.Wrap(errors.New("bwrap is not setuid root"), 0)
  }

  return nil
}

func doSetup(ctx *mansion.Context) {
	ctx.Must(Setup())
}

func Setup() error {
  fmt.Printf("Setup")

	nullConsumer := &state.Consumer{}

	err := Check(nullConsumer)
	if err == nil {
		fmt.Printf("Already set up properly!")
		return nil
	}

  executable, err := os.Executable()
  if err != nil {
    return errors.Wrap(err, 0)
  }

  firejailPath := filepath.Join(filepath.Dir(executable), "bin", "bwrap")
  err = os.Chown(firejailPath, 0, 0)
  if err != nil {
		return errors.Wrap(err, 0)
	}

  err = os.Chmod(firejailPath, 0755 | os.ModeSetuid)
  if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
