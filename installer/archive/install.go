package archive

import (
	"os"
	"path/filepath"

	"github.com/itchio/savior"

	"github.com/go-errors/errors"
	"github.com/modulesio/isolator/cmd/operate"
	"github.com/modulesio/isolator/installer"
	"github.com/modulesio/isolator/installer/archive/intervalsaveconsumer"
	"github.com/modulesio/isolator/installer/bfs"
)

func (m *Manager) Install(params *installer.InstallParams) (*installer.InstallResult, error) {
	consumer := params.Consumer

	var res = installer.InstallResult{
		Files: []string{},
	}

	archiveInfo := params.InstallerInfo.ArchiveInfo

	ex, err := archiveInfo.GetExtractor(params.File, consumer)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	ex.SetConsumer(consumer)

	statePath := filepath.Join(params.StageFolderPath, "install-state.dat")
	sc := intervalsaveconsumer.New(statePath, intervalsaveconsumer.DefaultInterval, consumer, params.Context)
	ex.SetSaveConsumer(sc)

	cancelled := false
	defer func() {
		if !cancelled {
			consumer.Infof("Clearing archive install state")
			os.Remove(statePath)
		}
	}()

	checkpoint := &savior.ExtractorCheckpoint{}
	err = sc.Load(checkpoint)
	if err != nil {
		consumer.Warnf("Could not load checkpoint, ignoring: %s", err.Error())
		checkpoint = nil
	}

	sink := &savior.FolderSink{
		Directory: params.InstallFolderPath,
		Consumer:  consumer,
	}

	aRes, err := ex.Resume(checkpoint, sink)
	if err != nil {
		if errors.Is(err, savior.ErrStop) {
			cancelled = true
			return nil, operate.ErrCancelled
		}
		return nil, errors.Wrap(err, 0)
	}

	err = sink.Close()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	for _, entry := range aRes.Entries {
		res.Files = append(res.Files, entry.CanonicalPath)
	}

	err = bfs.BustGhosts(&bfs.BustGhostsParams{
		Folder:   params.InstallFolderPath,
		NewFiles: res.Files,
		Receipt:  params.ReceiptIn,

		Consumer: params.Consumer,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return &res, nil
}
