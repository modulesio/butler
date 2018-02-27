package main

import (
	"github.com/modulesio/isolator/cmd/launch/launchers/html"
	"github.com/modulesio/isolator/cmd/launch/launchers/native"
	"github.com/modulesio/isolator/cmd/launch/launchers/shell"
	"github.com/modulesio/isolator/cmd/launch/launchers/url"
)

func init() {
	native.Register()
	shell.Register()
	html.Register()
	url.Register()
}
