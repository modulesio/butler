package installer

import (
	"io"
	"path/filepath"
	"time"

	"github.com/itchio/savior"

	"github.com/go-errors/errors"
	"github.com/modulesio/butler/archive"
	"github.com/modulesio/butler/configurator"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

func GetInstallerInfo(consumer *state.Consumer, file eos.File) (*InstallerInfo, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	target := stat.Name()
	ext := filepath.Ext(target)
	name := filepath.Base(target)

	consumer.Infof("↝ For source (%s)", name)

	if typ, ok := installerForExt[ext]; ok {
		if typ == InstallerTypeArchive {
			// let code flow, probe it as archive
		} else {
			consumer.Infof("✓ Using file extension registry (%s)", typ)
			return &InstallerInfo{
				Type: typ,
			}, nil
		}
	}

	// configurator is what we do first because it's generally fast:
	// it shouldn't read *much* of the remote file, and with httpfile
	// caching, it's even faster. whereas 7-zip might read a *bunch*
	// of an .exe file before it gives up

	consumer.Infof("  Probing with configurator...")

	beforeConfiguratorProbe := time.Now()
	candidate, err := configurator.Sniff(file, target, stat.Size())
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	consumer.Debugf("  (took %s)", time.Since(beforeConfiguratorProbe))

	var typePerConfigurator = InstallerTypeUnknown

	if candidate != nil {
		consumer.Infof("  Candidate: %s", candidate.String())
		typePerConfigurator = getInstallerTypeForCandidate(consumer, candidate)
	} else {
		consumer.Infof("  No results from configurator")
	}

	if typePerConfigurator == InstallerTypeUnknown || typePerConfigurator == InstallerTypeNaked || typePerConfigurator == InstallerTypeArchive {
		// some archive types are better sniffed by 7-zip and/or butler's own
		// decompression engines, so if configurator returns naked, we try
		// to open as an archive.
		beforeArchiveProbe := time.Now()
		consumer.Infof("  Probing as archive...")

		// seek to start first because configurator may have seeked itself
		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		archiveInfo, err := archive.Probe(&archive.TryOpenParams{
			File:     file,
			Consumer: consumer,
		})
		consumer.Debugf("  (took %s)", time.Since(beforeArchiveProbe))
		if err == nil {
			consumer.Infof("✓ Source is a supported archive format (%s)", archiveInfo.Format)
			if archiveInfo.Features.ResumeSupport == savior.ResumeSupportNone {
				// TODO: force downloading to disk first for those
				consumer.Warnf("    ...but this format has no/poor resume support, interruptions will waste network/CPU time")
			}

			return &InstallerInfo{
				Type:        InstallerTypeArchive,
				ArchiveInfo: archiveInfo,
			}, nil
		}

		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	consumer.Infof("✓ Using configurator results")
	return &InstallerInfo{
		Type: typePerConfigurator,
	}, nil
}

func getInstallerTypeForCandidate(consumer *state.Consumer, candidate *configurator.Candidate) InstallerType {
	switch candidate.Flavor {

	case configurator.FlavorNativeWindows:
		if candidate.WindowsInfo != nil && candidate.WindowsInfo.InstallerType != "" {
			typ := (InstallerType)(candidate.WindowsInfo.InstallerType)
			consumer.Infof("  → Windows installer of type %s", typ)
			return typ
		}

		consumer.Infof("  → Native windows executable, but not an installer")
		return InstallerTypeNaked

	case configurator.FlavorNativeMacos:
		consumer.Infof("  → Native macOS executable")
		return InstallerTypeNaked

	case configurator.FlavorNativeLinux:
		consumer.Infof("  → Native linux executable")
		return InstallerTypeNaked

	case configurator.FlavorScript:
		consumer.Infof("  → Script")
		if candidate.ScriptInfo != nil && candidate.ScriptInfo.Interpreter != "" {
			consumer.Infof("    with interpreter %s", candidate.ScriptInfo.Interpreter)
		}
		return InstallerTypeNaked

	case configurator.FlavorScriptWindows:
		consumer.Infof("  → Windows script")
		return InstallerTypeNaked
	}

	return InstallerTypeUnknown
}

func IsWindowsInstaller(typ InstallerType) bool {
	switch typ {
	case InstallerTypeMSI:
		return true
	case InstallerTypeNsis:
		return true
	case InstallerTypeInno:
		return true
	default:
		return false
	}
}
