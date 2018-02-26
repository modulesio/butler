package mansion

import (
	"context"
	"fmt"

	"github.com/go-errors/errors"
	"github.com/modulesio/butler/comm"
	"github.com/itchio/wharf/pwr"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type DoCommand func(ctx *Context)

type Context struct {
	App      *kingpin.Application
	Commands map[string]DoCommand

	// Identity is the path to the credentials file
	Identity string

	// Address is the URL of the itch.io API server we're talking to
	Address string

	// VersionString is the complete version string
	VersionString string

	// Version is just the version number, as a string
	Version string

	// Quiet silences all output
	Quiet bool

	// Verbose enables chatty output
	Verbose bool

	CompressionAlgorithm string
	CompressionQuality   int

	Cancelled bool
}

func NewContext(app *kingpin.Application) *Context {
	return &Context{
		App:      app,
		Commands: make(map[string]DoCommand),
	}
}

func (ctx *Context) Register(clause *kingpin.CmdClause, do DoCommand) {
	ctx.Commands[clause.FullCommand()] = do
}

func (ctx *Context) Must(err error) {
	if err != nil {
		switch err := err.(type) {
		case *errors.Error:
			comm.Die(err.ErrorStack())
		default:
			comm.Die(err.Error())
		}
	}
}

func (ctx *Context) UserAgent() string {
	return fmt.Sprintf("butler/%s", ctx.VersionString)
}

func (ctx *Context) CompressionSettings() pwr.CompressionSettings {
	var algo pwr.CompressionAlgorithm

	switch ctx.CompressionAlgorithm {
	case "none":
		algo = pwr.CompressionAlgorithm_NONE
	case "brotli":
		algo = pwr.CompressionAlgorithm_BROTLI
	case "gzip":
		algo = pwr.CompressionAlgorithm_GZIP
	case "zstd":
		algo = pwr.CompressionAlgorithm_ZSTD
	default:
		panic(fmt.Errorf("Unknown compression algorithm: %s", algo))
	}

	return pwr.CompressionSettings{
		Algorithm: algo,
		Quality:   int32(ctx.CompressionQuality),
	}
}

func (ctx *Context) Context() context.Context {
	return context.Background()
}
