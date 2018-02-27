package fetch

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-errors/errors"
	"github.com/modulesio/isolator/comm"
	"github.com/modulesio/isolator/mansion"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/wharf/archiver"
	"github.com/itchio/wharf/eos"
)

var args = struct {
	target *string
	out    *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("fetch", "Download and extract the latest build of a channel from itch.io")
	ctx.Register(cmd, do)

	args.target = cmd.Arg("target", "Which user/project:channel to fetch from, for example 'leafo/x-moon:win-64'. Targets are of the form project:channel where project is username/game or game_id.").Required().String()
	args.out = cmd.Arg("out", "Directory to fetch and extract build to").Required().String()
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, *args.target, *args.out))
}

func Do(ctx *mansion.Context, specStr string, outPath string) error {
	err := os.MkdirAll(outPath, os.FileMode(0755))
	if err != nil {
		return errors.Wrap(err, 1)
	}

	outFiles, err := ioutil.ReadDir(outPath)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	if len(outFiles) > 0 {
		return fmt.Errorf("Destination directory %s exists and is not empty", outPath)
	}

	spec, err := itchio.ParseSpec(specStr)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = spec.EnsureChannel()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	client, err := ctx.AuthenticateViaOauth()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	comm.Opf("Getting last build of channel %s", spec.Channel)

	channelResponse, err := client.GetChannel(spec.Target, spec.Channel)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	if channelResponse.Channel.Head == nil {
		return fmt.Errorf("Channel %s doesn't have any builds yet", spec.Channel)
	}

	head := *channelResponse.Channel.Head
	var headArchive *itchio.BuildFile

	for _, file := range head.Files {
		comm.Debugf("found file %v", file)
		if file.Type == itchio.BuildFileTypeArchive && file.SubType == itchio.BuildFileSubTypeDefault && file.State == itchio.BuildFileStateUploaded {
			headArchive = file
			break
		}
	}

	if headArchive == nil {
		return fmt.Errorf("Channel %s's latest build is still processing", spec.Channel)
	}

	url := head.ItchfsURL(headArchive, client.Key)

	remoteFile, err := eos.Open(url)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	stats, err := remoteFile.Stat()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	settings := archiver.ExtractSettings{
		Consumer: comm.NewStateConsumer(),
		OnUncompressedSizeKnown: func(totalBytes int64) {
			comm.StartProgressWithTotalBytes(totalBytes)
		},
	}

	comm.Opf("Extracting into %s", outPath)
	result, err := archiver.Extract(remoteFile, stats.Size(), outPath, settings)
	comm.EndProgress()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	comm.Statf("Extracted %d dirs, %d files, %d links into %s", result.Dirs, result.Files, result.Symlinks, outPath)

	if err != nil {
		return errors.Wrap(err, 1)
	}
	return nil
}
