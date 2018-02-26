package main

import (
	"github.com/modulesio/butler/installer/archive"
	"github.com/modulesio/butler/installer/inno"
	"github.com/modulesio/butler/installer/msi"
	"github.com/modulesio/butler/installer/naked"
	"github.com/modulesio/butler/installer/nsis"
)

func init() {
	naked.Register()
	archive.Register()
	nsis.Register()
	inno.Register()
	msi.Register()
}
