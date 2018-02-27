package ls

import (
	"archive/tar"
	"encoding/binary"
	"io"
	"os"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/arkive/zip"
	"github.com/modulesio/isolator/comm"
	"github.com/modulesio/isolator/mansion"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

var args = struct {
	file *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("ls", "Prints the list of files, dirs and symlinks contained in a patch file, signature file, or archive")
	args.file = cmd.Arg("file", "A file you'd like to list the contents of").Required().String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, *args.file))
}

func Do(ctx *mansion.Context, inPath string) error {
	reader, err := eos.Open(inPath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	path := eos.Redact(inPath)

	defer reader.Close()

	stats, err := reader.Stat()
	if os.IsNotExist(err) {
		comm.Dief("%s: no such file or directory", path)
	}
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if stats.IsDir() {
		comm.Logf("%s: directory", path)
		return nil
	}

	if stats.Size() == 0 {
		comm.Logf("%s: empty file. peaceful.", path)
		return nil
	}

	log := func(line string) {
		comm.Logf(line)
	}

	source := seeksource.FromFile(reader)

	_, err = source.Resume(nil)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	var magic int32
	err = binary.Read(source, wire.Endianness, &magic)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	switch magic {
	case pwr.PatchMagic:
		{
			h := &pwr.PatchHeader{}
			rctx := wire.NewReadContext(source)
			err = rctx.ReadMessage(h)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			rctx, err = pwr.DecompressWire(rctx, h.GetCompression())
			if err != nil {
				return errors.Wrap(err, 0)
			}
			container := &tlc.Container{}
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			log("pre-patch container:")
			container.Print(log)

			container.Reset()
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			log("================================")
			log("post-patch container:")
			container.Print(log)
		}

	case pwr.SignatureMagic:
		{
			h := &pwr.SignatureHeader{}
			rctx := wire.NewReadContext(source)
			err := rctx.ReadMessage(h)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			rctx, err = pwr.DecompressWire(rctx, h.GetCompression())
			if err != nil {
				return errors.Wrap(err, 0)
			}
			container := &tlc.Container{}
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.Wrap(err, 0)
			}
			container.Print(log)
		}

	case pwr.ManifestMagic:
		{
			h := &pwr.ManifestHeader{}
			rctx := wire.NewReadContext(source)
			err := rctx.ReadMessage(h)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			rctx, err = pwr.DecompressWire(rctx, h.GetCompression())
			if err != nil {
				return errors.Wrap(err, 0)
			}

			container := &tlc.Container{}
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.Wrap(err, 0)
			}
			container.Print(log)
		}

	case pwr.WoundsMagic:
		{
			wh := &pwr.WoundsHeader{}
			rctx := wire.NewReadContext(source)
			err := rctx.ReadMessage(wh)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			container := &tlc.Container{}
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.Wrap(err, 0)
			}
			container.Print(log)

			for {
				wound := &pwr.Wound{}
				err = rctx.ReadMessage(wound)
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					} else {
						return errors.Wrap(err, 0)
					}
				}
				comm.Logf(wound.PrettyString(container))
			}
		}

	default:
		_, err := reader.Seek(0, io.SeekStart)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		wasZip := func() bool {
			zr, err := zip.NewReader(reader, stats.Size())
			if err != nil {
				if err != zip.ErrFormat {
					ctx.Must(err)
				}
				return false
			}

			container, err := tlc.WalkZip(zr, &tlc.WalkOpts{
				Filter: func(fi os.FileInfo) bool { return true },
			})
			ctx.Must(err)
			container.Print(log)
			return true
		}()

		if wasZip {
			return nil
		}

		wasTar := func() bool {
			tr := tar.NewReader(reader)

			for {
				hdr, err := tr.Next()
				if err != nil {
					if err == io.EOF {
						break
					}
					return false
				}

				comm.Logf("%s %10s %s", os.FileMode(hdr.Mode), humanize.IBytes(uint64(hdr.Size)), hdr.Name)
			}
			return true
		}()

		if wasTar {
			return nil
		}

		comm.Logf("%s: not able to list contents", path)
	}

	return nil
}
