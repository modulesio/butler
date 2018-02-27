// +build windows

// This package implements a sandbox for Windows. It works by
// creating a less-privileged user, `isolator-user-XXXXX`, which
// we hide from login and share a game's folder before we launch
// it (then unshare it immediately after).
//
// If you want to see/manage the user the sandbox creates,
// you can use "lusrmgr.msc" on Windows (works in Win+R)
package winsandbox

import (
	"fmt"
	"syscall"
	"time"

	"github.com/modulesio/isolator/runner/syscallex"

	"github.com/go-errors/errors"
	"github.com/modulesio/isolator/comm"
	"github.com/modulesio/isolator/runner/winutil"
	"github.com/itchio/wharf/state"

	"github.com/modulesio/isolator/mansion"
)

var setfilepermissionsArgs = struct {
	file    *string
	change  *string
	rights  *string
	trustee *string
	inherit *bool
}{}

var checkAccessArgs = struct {
	file *string
}{}

func Register(ctx *mansion.Context) {
	parentCmd := ctx.App.Command("winsandbox", "Use or manage the itch.io sandbox for Windows").Hidden()

	{
		cmd := parentCmd.Command("check", "Verify that the sandbox is properly set up").Hidden()
		ctx.Register(cmd, doCheck)
	}

	{
		cmd := parentCmd.Command("setup", "Set up the sandbox (requires elevation)").Hidden()
		ctx.Register(cmd, doSetup)
	}

	{
		cmd := parentCmd.Command("setfilepermissions", "Set up the sandbox (requires elevation)").Hidden()
		setfilepermissionsArgs.file = cmd.Arg("file", "Name of file (or directory) to manipulate").Required().String()
		setfilepermissionsArgs.change = cmd.Arg("change", "Operation").Required().Enum("grant", "revoke")
		setfilepermissionsArgs.rights = cmd.Arg("rights", "Rights to grant/revoke").Required().Enum("read", "write", "execute", "all", "full")
		setfilepermissionsArgs.trustee = cmd.Arg("trustee", "Name of trustee").Required().String()
		setfilepermissionsArgs.inherit = cmd.Flag("inherit", "Whether to inherit").Required().Bool()
		ctx.Register(cmd, doSetfilepermissions)
	}

	{
		cmd := parentCmd.Command("checkaccess", "Check if the sandbox user has access to a certain file").Hidden()
		checkAccessArgs.file = cmd.Arg("file", "Name of file (or directory) to check access for").Required().String()
		ctx.Register(cmd, doCheckAccess)
	}
}

func doCheck(ctx *mansion.Context) {
	ctx.Must(Check(comm.NewStateConsumer()))
}

func Check(consumer *state.Consumer) error {
	consumer.Opf("Retrieving player data from registry...")
	pd, err := GetPlayerData()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Statf("Sandbox user is (%s)", pd.Username)

	consumer.Opf("Trying to log in...")

	token, err := winutil.Logon(pd.Username, ".", pd.Password)

	if err != nil {
		rescued := false

		if en, ok := winutil.AsErrno(err); ok {
			switch en {
			case syscallex.ERROR_PASSWORD_EXPIRED:
			case syscallex.ERROR_PASSWORD_MUST_CHANGE:
				// Some Windows versions (10 for example) expire password automatically.
				// Thankfully, we can renew it without administrator access, simply by using the old one.
				consumer.Opf("Password has expired, setting new password...")
				newPassword := generatePassword()

				err := syscallex.NetUserChangePassword(
					nil, // domainname
					syscall.StringToUTF16Ptr(pd.Username),
					syscall.StringToUTF16Ptr(pd.Password),
					syscall.StringToUTF16Ptr(newPassword),
				)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				pd.Password = newPassword
				err = pd.Save()
				if err != nil {
					return errors.Wrap(err, 0)
				}

				token, err = winutil.Logon(pd.Username, ".", pd.Password)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				consumer.Statf("Set new password successfully!")

				rescued = true
			}
		}

		if !rescued {
			return errors.Wrap(err, 0)
		}
	}
	defer syscall.CloseHandle(token)

	consumer.Statf("Everything looks good!")

	return nil
}

func doSetup(ctx *mansion.Context) {
	ctx.Must(Setup())
}

func Setup() error {
	startTime := time.Now()

  fmt.Printf("Setup")

	nullConsumer := &state.Consumer{}

	err := Check(nullConsumer)
	if err == nil {
		fmt.Printf("Already set up properly!")
		return nil
	}

	username := fmt.Sprintf("isolator-user-%x", time.Now().Unix())
  fmt.Printf("Generated username (%s)", username)

	password := generatePassword()
	fmt.Printf("Generated password (%s)", password)

	comment := "isolator sandbox user"

	fmt.Printf("Adding user...")

	err = winutil.AddUser(username, password, comment)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	fmt.Printf("Removing from Users group (so it doesn't show up as a login option)...")

	err = winutil.RemoveUserFromUsersGroup(username)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	fmt.Printf("Loading profile for the first time (to create some directories)...")

	err = winutil.LoadProfileOnce(username, ".", password)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	fmt.Printf("Saving to credentials registry...")

	pd := &PlayerData{
		Username: username,
		Password: password,
	}
	err = pd.Save()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	fmt.Printf("All done! (in %s)", time.Since(startTime))

	return nil
}

func doSetfilepermissions(ctx *mansion.Context) {
	ctx.Must(Setfilepermissions(comm.NewStateConsumer()))
}

func Setfilepermissions(consumer *state.Consumer) error {
	entry := &winutil.ShareEntry{
		Path: *setfilepermissionsArgs.file,
	}

	if *setfilepermissionsArgs.inherit {
		entry.Inheritance = winutil.InheritanceModeFull
	} else {
		entry.Inheritance = winutil.InheritanceModeNone
	}

	switch *setfilepermissionsArgs.rights {
	case "read":
		entry.Rights = winutil.RightsRead
	case "write":
		entry.Rights = winutil.RightsWrite
	case "execute":
		entry.Rights = winutil.RightsExecute
	case "all":
		entry.Rights = winutil.RightsAll
	case "full":
		entry.Rights = winutil.RightsFull
	default:
		return fmt.Errorf("unknown rights: %s", *setfilepermissionsArgs.rights)
	}

	policy := &winutil.SharingPolicy{
		Trustee: *setfilepermissionsArgs.trustee,
		Entries: []*winutil.ShareEntry{entry},
	}

	switch *setfilepermissionsArgs.change {
	case "grant":
		consumer.Opf("Granting %s", policy)
		err := policy.Grant()
		if err != nil {
			return errors.Wrap(err, 0)
		}
	case "revoke":
		consumer.Opf("Revoking %s", policy)
		err := policy.Revoke()
		if err != nil {
			return errors.Wrap(err, 0)
		}
	default:
		return fmt.Errorf("unknown change: %s", *setfilepermissionsArgs.change)
	}

	comm.Statf("Policy applied successfully")

	return nil
}

func doCheckAccess(ctx *mansion.Context) {
	ctx.Must(CheckAccess(comm.NewStateConsumer()))
}

type checkAccessSpec struct {
	name  string
	flags uint32
}

var checkAccessSpecs = []checkAccessSpec{
	checkAccessSpec{"read", syscallex.GENERIC_READ},
	checkAccessSpec{"write", syscallex.GENERIC_WRITE},
	checkAccessSpec{"execute", syscallex.GENERIC_EXECUTE},
	checkAccessSpec{"all", syscallex.GENERIC_ALL},
}

func CheckAccess(consumer *state.Consumer) error {
	pd, err := GetPlayerData()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	impersonationToken, err := winutil.GetImpersonationToken(pd.Username, ".", pd.Password)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer winutil.SafeRelease(uintptr(impersonationToken))

	for _, spec := range checkAccessSpecs {
		hasAccess, err := winutil.UserHasPermission(
			impersonationToken,
			spec.flags,
			*checkAccessArgs.file,
		)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if hasAccess {
			comm.Opf("User has %s access", spec.name)
		} else {
			comm.Opf("User does not have %s access", spec.name)
		}
	}

	return nil
}
