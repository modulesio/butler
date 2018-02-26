package operate

import (
	"github.com/go-errors/errors"
	"github.com/modulesio/butler/buse"
	"github.com/modulesio/butler/cmd/wipe"
	"github.com/modulesio/butler/installer"
	"github.com/modulesio/butler/installer/bfs"
)

func uninstall(oc *OperationContext, meta *MetaSubcontext) error {
	consumer := oc.Consumer()

	params := meta.data.UninstallParams

	if params == nil {
		return errors.New("Missing uninstall params")
	}

	if params.InstallFolder == "" {
		return errors.New("Missing install folder in uninstall")
	}

	consumer.Infof("→ Uninstalling %s", params.InstallFolder)

	var installerType = installer.InstallerTypeUnknown

	receipt, err := bfs.ReadReceipt(params.InstallFolder)
	if err != nil {
		consumer.Warnf("Could not read receipt: %s", err.Error())
	}

	if receipt != nil && receipt.InstallerName != "" {
		installerType = (installer.InstallerType)(receipt.InstallerName)
	}

	consumer.Infof("Will use installer %s", installerType)
	manager := installer.GetManager(string(installerType))
	if manager == nil {
		consumer.Warnf("No manager for installer %s", installerType)
		consumer.Infof("Falling back to archive")

		manager = installer.GetManager("archive")
		if manager == nil {
			return errors.New("archive install manager not found, can't uninstall")
		}
	}

	managerUninstallParams := &installer.UninstallParams{
		InstallFolderPath: params.InstallFolder,
		Consumer:          consumer,
		Receipt:           receipt,
	}

	err = oc.conn.Notify(oc.ctx, "TaskStarted", &buse.TaskStartedNotification{
		Reason: buse.TaskReasonUninstall,
		Type:   buse.TaskTypeUninstall,
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	oc.StartProgress()
	err = manager.Uninstall(managerUninstallParams)
	oc.EndProgress()

	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = oc.conn.Notify(oc.ctx, "TaskSucceeded", &buse.TaskSucceededNotification{
		Type: buse.TaskTypeUninstall,
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = wipe.Do(consumer, params.InstallFolder)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
