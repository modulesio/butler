package main

import (
	"github.com/modulesio/butler/cmd/launch/launchers/html"
	"github.com/modulesio/butler/cmd/launch/launchers/native"
	"github.com/modulesio/butler/cmd/launch/launchers/shell"
	"github.com/modulesio/butler/cmd/launch/launchers/url"
)

func init() {
	native.Register()
	shell.Register()
	html.Register()
	url.Register()
}
