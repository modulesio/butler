package html

import (
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/modulesio/isolator/buse"
	"github.com/modulesio/isolator/cmd/launch"
)

func Register() {
	launch.Register(launch.LaunchStrategyHTML, &Launcher{})
}

type Launcher struct{}

var _ launch.Launcher = (*Launcher)(nil)

func (l *Launcher) Do(params *launch.LauncherParams) error {
	ctx := params.Ctx
	conn := params.Conn

	rootFolder := params.InstallFolder
	indexPath, err := filepath.Rel(rootFolder, params.FullTargetPath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	var r buse.HTMLLaunchResult
	err = conn.Call(ctx, "HTMLLaunch", &buse.HTMLLaunchParams{
		RootFolder: rootFolder,
		IndexPath:  indexPath,
		Args:       params.Args,
		Env:        params.Env,
	}, &r)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
