package walk

import (
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/modulesio/butler/comm"
	"github.com/modulesio/butler/mansion"
	"github.com/itchio/wharf/tlc"
)

var args = struct {
	dir         *string
	dereference *bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("walk", "Finds all files in a directory").Hidden()
	args.dir = cmd.Arg("dir", "A dir you want to walk").Required().String()
	args.dereference = cmd.Flag("dereference", "Follow symlinks").Default("false").Bool()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, *args.dir, *args.dereference))
}

func Do(ctx *mansion.Context, dir string, dereference bool) error {
	startTime := time.Now()

	container, err := tlc.WalkDir(dir, &tlc.WalkOpts{
		Filter:      func(fi os.FileInfo) bool { return true },
		Dereference: dereference,
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	totalEntries := 0
	send := func(path string) {
		totalEntries++
		comm.ResultOrPrint(&mansion.WalkResult{
			Type: "entry",
			Path: path,
		}, func() {
			comm.Logf("- %s", path)
		})
	}

	for _, f := range container.Files {
		send(f.Path)
	}

	for _, s := range container.Symlinks {
		send(s.Path)
	}

	comm.ResultOrPrint(&mansion.WalkResult{
		Type: "totalSize",
		Size: container.Size,
	}, func() {
		comm.Statf("%d entries (%s) walked in %s", totalEntries, humanize.IBytes(uint64(container.Size)), time.Since(startTime))
	})

	return nil
}
