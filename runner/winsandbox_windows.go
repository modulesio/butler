package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modulesio/isolator/installer"

	// "github.com/modulesio/isolator/buse"
	"github.com/modulesio/isolator/cmd/elevate"
	"github.com/modulesio/isolator/cmd/operate"

	"github.com/go-errors/errors"
	"github.com/modulesio/isolator/cmd/winsandbox"
	"github.com/modulesio/isolator/comm"
	"github.com/modulesio/isolator/runner/execas"
	"github.com/modulesio/isolator/runner/syscallex"
	"github.com/modulesio/isolator/runner/winutil"
	"github.com/itchio/wharf/state"
)

type winsandboxRunner struct {
	params *RunnerParams

	playerData *winsandbox.PlayerData
}

var _ Runner = (*winsandboxRunner)(nil)

func newWinSandboxRunner(params *RunnerParams) (Runner, error) {
	wr := &winsandboxRunner{
		params: params,
	}
	return wr, nil
}

func (wr *winsandboxRunner) Prepare() error {

	nullConsumer := &state.Consumer{}
	err := winsandbox.Check(nullConsumer)
	if err != nil {
		res, err := installer.RunSelf(&installer.RunSelfParams{
			Consumer: nullConsumer,
			Args: []string{
				"--elevate",
				"winsandbox",
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

		err = winsandbox.Check(nullConsumer)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	playerData, err := winsandbox.GetPlayerData()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	wr.playerData = playerData

  err = os.MkdirAll(filepath.Join(wr.params.Dir, ".isolator"), 0755)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func (wr *winsandboxRunner) Run() error {
	var err error
	params := wr.params
	pd := wr.playerData

	env, err := wr.getEnvironment()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	sp, err := wr.getSharingPolicy()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = sp.Grant()
	if err != nil {
		comm.Warnf(err.Error())
		comm.Warnf("Attempting launch anyway...")
	}

	defer sp.Revoke()

	err = SetupJobObject()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	cmd := execas.Command(params.FullTargetPath, params.Args...)
	cmd.Username = pd.Username
	cmd.Domain = "."
	cmd.Password = pd.Password
	cmd.Dir = params.Dir
	cmd.Env = env
	cmd.Stdin = params.Stdin
	cmd.Stdout = params.Stdout
	cmd.Stderr = params.Stderr
	cmd.SysProcAttr = &syscallex.SysProcAttr{
    LogonFlags: syscallex.LOGON_WITH_PROFILE,
    // CreationFlags: 0x08000000,
    // HideWindow: true,
	}

	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = WaitJobObject()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func (wr *winsandboxRunner) getSharingPolicy() (*winutil.SharingPolicy, error) {
	params := wr.params
	pd := wr.playerData

	sp := &winutil.SharingPolicy{
		Trustee: pd.Username,
	}

	impersonationToken, err := winutil.GetImpersonationToken(pd.Username, ".", pd.Password)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	defer winutil.SafeRelease(uintptr(impersonationToken))

  // Dir
	hasAccess, err := winutil.UserHasPermission(
		impersonationToken,
		syscallex.GENERIC_ALL,
		params.Dir,
	)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	if !hasAccess {
		sp.Entries = append(sp.Entries, &winutil.ShareEntry{
			Path:        params.Dir,
			Inheritance: winutil.InheritanceModeFull,
			Rights:      winutil.RightsFull,
		})
	}
  isolatorPath := filepath.Join(params.Dir, ".isolator")
  hasAccess, err = winutil.UserHasPermission(
		impersonationToken,
		syscallex.GENERIC_ALL,
		isolatorPath,
	)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	if !hasAccess {
		sp.Entries = append(sp.Entries, &winutil.ShareEntry{
			Path:        isolatorPath,
			Inheritance: winutil.InheritanceModeFull,
			Rights:      winutil.RightsFull,
		})
	}
	// cf. https://github.com/itchio/itch/issues/1470
	current := filepath.Dir(params.Dir)
	for i := 0; i < 128; i++ { // dumb failsafe
		hasAccess, err := winutil.UserHasPermission(
			impersonationToken,
			syscallex.GENERIC_READ,
			current,
		)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		if !hasAccess {
			sp.Entries = append(sp.Entries, &winutil.ShareEntry{
				Path:        current,
				Inheritance: winutil.InheritanceModeNone,
				Rights:      winutil.RightsRead,
			})
		}
		next := filepath.Dir(current)
		if next == current {
			break
		}
		current = next
	}

  /* // InstallFolder
	hasAccess, err = winutil.UserHasPermission(
		impersonationToken,
		syscallex.GENERIC_ALL,
		params.InstallFolder,
	)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	if !hasAccess {
		sp.Entries = append(sp.Entries, &winutil.ShareEntry{
			Path:        params.InstallFolder,
			Inheritance: winutil.InheritanceModeFull,
			Rights:      winutil.RightsFull,
		})
	}
	// cf. https://github.com/itchio/itch/issues/1470
	current = filepath.Dir(params.InstallFolder)
	for i := 0; i < 128; i++ { // dumb failsafe
		hasAccess, err := winutil.UserHasPermission(
			impersonationToken,
			syscallex.GENERIC_READ,
			current,
		)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		if !hasAccess {
			sp.Entries = append(sp.Entries, &winutil.ShareEntry{
				Path:        current,
				Inheritance: winutil.InheritanceModeNone,
				Rights:      winutil.RightsRead,
			})
		}
		next := filepath.Dir(current)
		if next == current {
			break
		}
		current = next
	} */

	return sp, nil
}

func (wr *winsandboxRunner) getEnvironment() ([]string, error) {
	params := wr.params
	pd := wr.playerData

	env := params.Env
	setEnv := func(key string, value string) {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	setEnv("username", pd.Username)
	// we're not setting `userdomain` or `userdomain_roaming_profile`,
	// since we expect those to be the same for the regular user
	// and the sandbox user

	err := winutil.Impersonate(pd.Username, ".", pd.Password, func() error {
		profileDir, err := winutil.GetFolderPath(winutil.FolderTypeProfile)
		if err != nil {
			return errors.Wrap(err, 0)
		}
		// environment variables are case-insensitive on windows,
		// and exec{,as}.Command do case-insensitive deduplication properly
		setEnv("userprofile", profileDir)

		// when %userprofile% is `C:\Users\terry`,
		// %homepath% is usually `\Users\terry`.
		homePath := strings.TrimPrefix(profileDir, filepath.VolumeName(profileDir))
		setEnv("homepath", homePath)

		appDataDir, err := winutil.GetFolderPath(winutil.FolderTypeAppData)
		if err != nil {
			return errors.Wrap(err, 0)
		}
		setEnv("appdata", appDataDir)

		localAppDataDir, err := winutil.GetFolderPath(winutil.FolderTypeLocalAppData)
		if err != nil {
			return errors.Wrap(err, 0)
		}
		setEnv("localappdata", localAppDataDir)

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return env, nil
}
