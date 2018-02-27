package main

import (
	/* "github.com/modulesio/isolator/cmd/apply"
	"github.com/modulesio/isolator/cmd/apply2"
	"github.com/modulesio/isolator/cmd/auditzip"
	"github.com/modulesio/isolator/cmd/clean"
	"github.com/modulesio/isolator/cmd/configure"
	"github.com/modulesio/isolator/cmd/cp"
	"github.com/modulesio/isolator/cmd/diff"
	"github.com/modulesio/isolator/cmd/ditto"
	"github.com/modulesio/isolator/cmd/dl" */
	"github.com/modulesio/isolator/cmd/elevate"
	/* "github.com/modulesio/isolator/cmd/elfprops"
	"github.com/modulesio/isolator/cmd/exeprops"
	"github.com/modulesio/isolator/cmd/extract"
	"github.com/modulesio/isolator/cmd/fetch"
	"github.com/modulesio/isolator/cmd/file"
	"github.com/modulesio/isolator/cmd/heal"
	"github.com/modulesio/isolator/cmd/launch"
	"github.com/modulesio/isolator/cmd/login"
	"github.com/modulesio/isolator/cmd/logout"
	"github.com/modulesio/isolator/cmd/ls"
	"github.com/modulesio/isolator/cmd/mkdir"
	"github.com/modulesio/isolator/cmd/msi" */
	"github.com/modulesio/isolator/cmd/pipe"
	/* "github.com/modulesio/isolator/cmd/prereqs"
	"github.com/modulesio/isolator/cmd/probe"
	"github.com/modulesio/isolator/cmd/push"
	"github.com/modulesio/isolator/cmd/repack" */
	"github.com/modulesio/isolator/cmd/run"
	"github.com/modulesio/isolator/cmd/runner"
	/* "github.com/modulesio/isolator/cmd/service"
	"github.com/modulesio/isolator/cmd/sign"
	"github.com/modulesio/isolator/cmd/sizeof"
	"github.com/modulesio/isolator/cmd/status"
	"github.com/modulesio/isolator/cmd/unsz"
	"github.com/modulesio/isolator/cmd/untar"
	"github.com/modulesio/isolator/cmd/unzip"
	"github.com/modulesio/isolator/cmd/upgrade"
	"github.com/modulesio/isolator/cmd/verify"
	"github.com/modulesio/isolator/cmd/version"
	"github.com/modulesio/isolator/cmd/walk"
	"github.com/modulesio/isolator/cmd/which" */
	"github.com/modulesio/isolator/cmd/linuxsandbox"
	"github.com/modulesio/isolator/cmd/winsandbox"
	/* "github.com/modulesio/isolator/cmd/wipe" */
	"github.com/modulesio/isolator/mansion"
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

	msi.Register(ctx)
	prereqs.Register(ctx) */

	/* extract.Register(ctx)
	unzip.Register(ctx)
	unsz.Register(ctx)
	untar.Register(ctx)
	auditzip.Register(ctx)

	repack.Register(ctx) */

	pipe.Register(ctx)
	elevate.Register(ctx)
	run.Register(ctx)
	runner.Register(ctx)

	/* exeprops.Register(ctx)
	elfprops.Register(ctx)

	configure.Register(ctx)

	apply2.Register(ctx)

	service.Register(ctx) */

	winsandbox.Register(ctx)
	linuxsandbox.Register(ctx)
}
