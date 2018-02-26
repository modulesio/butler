package unsz

import (
	"time"

	"github.com/itchio/savior"

	"github.com/modulesio/butler/archive/szextractor"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/modulesio/butler/comm"
	"github.com/modulesio/butler/mansion"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

var args = struct {
	file *string
	dir  *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("unsz", "Extract any archive file supported by 7-zip").Hidden()
	args.file = cmd.Arg("file", "Path of the archive to extract").Required().String()
	args.dir = cmd.Flag("dir", "An optional directory to which to extract files (defaults to CWD)").Default(".").Short('d').String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, &UnszParams{
		File: *args.file,
		Dir:  *args.dir,

		Consumer: comm.NewStateConsumer(),
	}))
}

type UnszParams struct {
	File string
	Dir  string

	Consumer *state.Consumer
}

func Do(ctx *mansion.Context, params *UnszParams) error {
	if params.File == "" {
		return errors.New("unsz: File must be specified")
	}
	if params.Dir == "" {
		return errors.New("unsz: Dir must be specified")
	}

	consumer := params.Consumer

	file, err := eos.Open(params.File)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer file.Close()

	stats, err := file.Stat()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Opf("Extracting %s to %s", stats.Name(), params.Dir)

	ex, err := szextractor.New(file, consumer)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	startTime := time.Now()

	sink := &savior.FolderSink{
		Directory: params.Dir,
	}

	comm.StartProgress()
	res, err := ex.Resume(nil, sink)
	comm.EndProgress()

	if err != nil {
		return errors.Wrap(err, 0)
	}

	duration := time.Since(startTime)
	bytesPerSec := float64(res.Size()) / duration.Seconds()
	consumer.Statf("Overall extraction speed: %s/s", humanize.IBytes(uint64(bytesPerSec)))

	return nil
}
