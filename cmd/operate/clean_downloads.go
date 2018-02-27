package operate

import (
	"io/ioutil"
	"os"
	"path/filepath"

	humanize "github.com/dustin/go-humanize"
	"github.com/modulesio/isolator/cmd/sizeof"
	"github.com/modulesio/isolator/cmd/wipe"
	"github.com/itchio/wharf/state"

	"github.com/modulesio/isolator/buse"
)

func CleanDownloadsSearch(params *buse.CleanDownloadsSearchParams, consumer *state.Consumer) (*buse.CleanDownloadsSearchResult, error) {
	// struct{} trick to use map as a set with 0-sized values
	whitemap := make(map[string]struct{})
	for _, whitelistPath := range params.Whitelist {
		whitemap[whitelistPath] = struct{}{}
	}

	var entries []*buse.CleanDownloadsEntry

	for _, root := range params.Roots {
		folders, err := ioutil.ReadDir(root)
		if err != nil {
			if os.IsNotExist(err) {
				// good, nothing to clean!
				continue
			} else {
				consumer.Warnf("Cannot scan root (%s): %s", root, err.Error())
				continue
			}
		}

		for _, folder := range folders {
			base := filepath.Base(folder.Name())
			absoluteFolderPath := filepath.Join(root, base)

			if _, ok := whitemap[base]; ok {
				// don't even consider it
				consumer.Debugf("Ignoring whitelisted (%s)")
				continue
			}

			// ey that's a candidate!
			folderSize, err := sizeof.Do(absoluteFolderPath)
			if err != nil {
				consumer.Warnf("Could not determine folder size: %s", err.Error())
			}

			entries = append(entries, &buse.CleanDownloadsEntry{
				Path: absoluteFolderPath,
				Size: folderSize,
			})
		}
	}

	res := &buse.CleanDownloadsSearchResult{
		Entries: entries,
	}
	return res, nil
}

func CleanDownloadsApply(params *buse.CleanDownloadsApplyParams, consumer *state.Consumer) (*buse.CleanDownloadsApplyResult, error) {
	for _, entry := range params.Entries {
		consumer.Infof("Wiping (%s) - %s", entry.Path, humanize.IBytes(uint64(entry.Size)))
		err := wipe.Do(consumer, entry.Path)
		if err != nil {
			consumer.Warnf("Could not wipe (%s): %s", entry.Path, err.Error())
		}
	}

	res := &buse.CleanDownloadsApplyResult{}
	return res, nil
}
