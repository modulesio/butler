package prereqs

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/modulesio/butler/manager"

	"github.com/go-errors/errors"
	"github.com/modulesio/butler/redist"
)

type PrereqAssessment struct {
	Done []string
	Todo []string
}

func (pc *PrereqsContext) AssessPrereqs(names []string) (*PrereqAssessment, error) {
	pa := &PrereqAssessment{}

	for _, name := range names {
		entry, err := pc.GetEntry(name)
		if entry == nil {
			pc.Consumer.Warnf("Prereq (%s) not found in registry, skipping...", name)
			continue
		}

		alreadyGood := false

		switch pc.Runtime.Platform {
		case manager.ItchPlatformWindows:
			alreadyGood, err = pc.AssessWindowsPrereq(name, entry)
		case manager.ItchPlatformLinux:
			alreadyGood, err = pc.AssessLinuxPrereq(name, entry)
		}

		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		if alreadyGood {
			// then it's already installed, cool!
			pa.Done = append(pa.Done, name)
			continue
		}

		pa.Todo = append(pa.Todo, name)
	}

	for _, name := range pa.Done {
		err := pc.MarkInstalled(name)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		continue
	}

	return pa, nil
}

func (pc *PrereqsContext) MarkerPath(name string) string {
	return filepath.Join(pc.PrereqsDir, name, ".installed")
}

func (pc *PrereqsContext) HasInstallMarker(name string) bool {
	path := pc.MarkerPath(name)
	_, err := os.Stat(path)
	return err == nil
}

func (pc *PrereqsContext) MarkInstalled(name string) error {
	if pc.HasInstallMarker(name) {
		// don't mark again
		return nil
	}

	contents := fmt.Sprintf("Installed on %s", time.Now())
	path := pc.MarkerPath(name)
	err := os.MkdirAll(filepath.Dir(path), os.FileMode(0755))
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = ioutil.WriteFile(path, []byte(contents), os.FileMode(0644))
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func (pc *PrereqsContext) AssessWindowsPrereq(name string, entry *redist.RedistEntry) (bool, error) {
	block := entry.Windows

	for _, registryKey := range block.RegistryKeys {
		if RegistryKeyExists(pc.Consumer, registryKey) {
			pc.Consumer.Debugf("Found registry key (%s)", registryKey)
			return true, nil
		}
	}

	return false, nil
}

func (pc *PrereqsContext) AssessLinuxPrereq(name string, entry *redist.RedistEntry) (bool, error) {
	block := entry.Linux

	switch block.Type {
	case redist.LinuxRedistTypeHosted:
		// cool!
	default:
		return false, fmt.Errorf("Don't know how to assess linux prereq of type (%s)", block.Type)
	}

	for _, sc := range block.SanityChecks {
		err := pc.RunSanityCheck(name, entry, sc)
		if err != nil {
			return false, nil
		}
	}

	return true, nil
}

func (pc *PrereqsContext) RunSanityCheck(name string, entry *redist.RedistEntry, sc *redist.LinuxSanityCheck) error {
	cmd := exec.Command(sc.Command, sc.Args...)
	cmd.Dir = pc.GetEntryDir(name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		pc.Consumer.Debugf("Sanity check failed:%s\n%s", err.Error(), string(output))
		return errors.Wrap(err, 0)
	}

	return nil
}
