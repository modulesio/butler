package which

import (
	"github.com/modulesio/butler/comm"
	"github.com/modulesio/butler/mansion"
	"github.com/kardianos/osext"
)

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("which", "Prints the path to this binary")
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	p, err := osext.Executable()
	ctx.Must(err)

	comm.Logf("You're running butler %s, from the following path:", ctx.VersionString)
	comm.Logf("%s", p)
}
