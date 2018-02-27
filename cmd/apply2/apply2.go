package apply2

import (
	"encoding/gob"
	"os"
	"path"

	"github.com/dchest/safefile"
	"github.com/go-errors/errors"
	"github.com/modulesio/isolator/comm"
	"github.com/modulesio/isolator/mansion"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr/bowl"
	"github.com/itchio/wharf/pwr/patcher"
	"github.com/itchio/wharf/state"
)

var args = struct {
	patch  *string
	dir    *string
	old    *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("apply2", "(Advanced) Use a patch to resumably patch a directory to a new version")
	args.patch = cmd.Arg("patch", "Patch file (.pwr), previously generated with the `diff` command.").Required().String()
	args.old = cmd.Arg("old", "Directory with old files").Required().String()
	args.dir = cmd.Flag("dir", "Directory for patched files and checkpoints").Required().String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(&Params{
		Patch:  *args.patch,
		Old:    *args.old,
		Dir:    *args.dir,
	}))
}

type Params struct {
	Patch  string
	Old    string
	Dir    string
}

func Do(params *Params) error {
	patch := params.Patch
	old := params.Old
	dir := params.Dir

	consumer := &state.Consumer{
		OnMessage: func(level string, message string) {
			comm.Logf("[%s] %s", level, message)
		},
	}

	patchReader, err := eos.Open(patch)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	patchSource := seeksource.FromFile(patchReader)
	_, err = patchSource.Resume(nil)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	p, err := patcher.New(patchSource, consumer)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	// comm.StartProgressWithTotalBytes(patchSource.Size())

	var checkpoint *patcher.Checkpoint
	checkpointPath := path.Join(dir, "checkpoint.bwl")

	checkpointFile, err := os.Open(checkpointPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrap(err, 0)
		}
	} else {
		defer checkpointFile.Close()

		checkpoint = &patcher.Checkpoint{}

		dec := gob.NewDecoder(checkpointFile)
		err := dec.Decode(checkpoint)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		// yay, we have a checkpoint!
	}

	p.SetSaveConsumer(&patcherSaveConsumer{
		shouldSave: func() bool {
			// TODO: patcher checkpoints are big. how often do we actually wanna do this?
			return true
		},
		save: func(c *patcher.Checkpoint) (patcher.AfterSaveAction, error) {
			checkpointFile, err := safefile.Create(checkpointPath, 0644)
			if err != nil {
				return patcher.AfterSaveStop, errors.Wrap(err, 0)
			}
			defer checkpointFile.Close()

			enc := gob.NewEncoder(checkpointFile)
			err = enc.Encode(c)
			if err != nil {
				return patcher.AfterSaveStop, errors.Wrap(err, 0)
			}

			err = checkpointFile.Commit()
			if err != nil {
				return patcher.AfterSaveStop, errors.Wrap(err, 0)
			}

			return patcher.AfterSaveContinue, nil
		},
	})

	targetPool := fspool.New(p.GetTargetContainer(), old)

	bowl, err := bowl.NewFreshBowl(&bowl.FreshBowlParams{
		SourceContainer: p.GetSourceContainer(),
		TargetContainer: p.GetTargetContainer(),
		TargetPool:      targetPool,
		OutputFolder:    dir,
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = p.Resume(checkpoint, targetPool, bowl)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	comm.EndProgress()

	return nil
}

type patcherSaveConsumer struct {
	shouldSave func() bool
	save       func(checkpoint *patcher.Checkpoint) (patcher.AfterSaveAction, error)
}

var _ patcher.SaveConsumer = (*patcherSaveConsumer)(nil)

func (psc *patcherSaveConsumer) ShouldSave() bool {
	return psc.shouldSave()
}

func (psc *patcherSaveConsumer) Save(checkpoint *patcher.Checkpoint) (patcher.AfterSaveAction, error) {
	return psc.save(checkpoint)
}
