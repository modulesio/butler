package main

import (
	"github.com/modulesio/isolator/installer/archive"
	"github.com/modulesio/isolator/installer/inno"
	"github.com/modulesio/isolator/installer/msi"
	"github.com/modulesio/isolator/installer/naked"
	"github.com/modulesio/isolator/installer/nsis"
)

func init() {
	naked.Register()
	archive.Register()
	nsis.Register()
	inno.Register()
	msi.Register()
}
