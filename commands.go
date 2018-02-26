package main

import (
	/* "github.com/modulesio/butler/cmd/apply"
	"github.com/modulesio/butler/cmd/apply2"
	"github.com/modulesio/butler/cmd/auditzip"
	"github.com/modulesio/butler/cmd/clean"
	"github.com/modulesio/butler/cmd/configure"
	"github.com/modulesio/butler/cmd/cp"
	"github.com/modulesio/butler/cmd/diff"
	"github.com/modulesio/butler/cmd/ditto"
	"github.com/modulesio/butler/cmd/dl" */
	"github.com/modulesio/butler/cmd/elevate"
	/* "github.com/modulesio/butler/cmd/elfprops"
	"github.com/modulesio/butler/cmd/exeprops"
	"github.com/modulesio/butler/cmd/extract"
	"github.com/modulesio/butler/cmd/fetch"
	"github.com/modulesio/butler/cmd/file"
	"github.com/modulesio/butler/cmd/heal"
	"github.com/modulesio/butler/cmd/launch"
	"github.com/modulesio/butler/cmd/login"
	"github.com/modulesio/butler/cmd/logout"
	"github.com/modulesio/butler/cmd/ls"
	"github.com/modulesio/butler/cmd/mkdir"
	"github.com/modulesio/butler/cmd/msi" */
	"github.com/modulesio/butler/cmd/pipe"
	"github.com/modulesio/butler/cmd/prereqs"
	/* "github.com/modulesio/butler/cmd/probe"
	"github.com/modulesio/butler/cmd/push"
	"github.com/modulesio/butler/cmd/repack"
	"github.com/modulesio/butler/cmd/run" */
	"github.com/modulesio/butler/cmd/runner"
	/* "github.com/modulesio/butler/cmd/service"
	"github.com/modulesio/butler/cmd/sign"
	"github.com/modulesio/butler/cmd/sizeof"
	"github.com/modulesio/butler/cmd/status"
	"github.com/modulesio/butler/cmd/unsz"
	"github.com/modulesio/butler/cmd/untar"
	"github.com/modulesio/butler/cmd/unzip"
	"github.com/modulesio/butler/cmd/upgrade"
	"github.com/modulesio/butler/cmd/verify"
	"github.com/modulesio/butler/cmd/version"
	"github.com/modulesio/butler/cmd/walk"
	"github.com/modulesio/butler/cmd/which" */
	"github.com/modulesio/butler/cmd/winsandbox"
	/* "github.com/modulesio/butler/cmd/wipe" */
	"github.com/modulesio/butler/mansion"
)

// Each of these specify their own arguments and flags in
// their own package.
func registerCommands(ctx *mansion.Context) {
	// documented commands

	/* login.Register(ctx)
	logout.Register(ctx)

	push.Register(ctx)
	fetch.Register(ctx)
	status.Register(ctx)

	file.Register(ctx)
	ls.Register(ctx)

	which.Register(ctx)
	version.Register(ctx)
	upgrade.Register(ctx)

	sign.Register(ctx)
	verify.Register(ctx)
	diff.Register(ctx)
	apply.Register(ctx)
	heal.Register(ctx)

	// hidden commands

	dl.Register(ctx)
	cp.Register(ctx)
	wipe.Register(ctx)
	sizeof.Register(ctx)
	mkdir.Register(ctx)
	ditto.Register(ctx)
	probe.Register(ctx)

	clean.Register(ctx)
	walk.Register(ctx)

	msi.Register(ctx) */
	prereqs.Register(ctx)

	/* extract.Register(ctx)
	unzip.Register(ctx)
	unsz.Register(ctx)
	untar.Register(ctx)
	auditzip.Register(ctx)

	repack.Register(ctx) */

	pipe.Register(ctx)
	elevate.Register(ctx)
	// run.Register(ctx)
	runner.Register(ctx)

	/* exeprops.Register(ctx)
	elfprops.Register(ctx)

	configure.Register(ctx)

	apply2.Register(ctx)

	service.Register(ctx) */

	winsandbox.Register(ctx)
}
