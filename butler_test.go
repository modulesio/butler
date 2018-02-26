package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"runtime"
	"strconv"
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/modulesio/butler/cmd/apply"
	"github.com/modulesio/butler/cmd/diff"
	"github.com/modulesio/butler/cmd/ditto"
	"github.com/modulesio/butler/cmd/sign"
	"github.com/modulesio/butler/comm"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"
	"github.com/stretchr/testify/assert"
)

// reverse must
func mist(t *testing.T, err error) {
	if err != nil {
		if se, ok := err.(*errors.Error); ok {
			t.Logf("Full stack: %s", se.ErrorStack())
		}
		assert.NoError(t, err)
		t.FailNow()
	}
}

func octal(perm os.FileMode) string {
	return strconv.FormatInt(int64(perm), 8)
}

func permFor(t *testing.T, path string) os.FileMode {
	t.Logf("Getting perm of %s", path)
	stat, err := os.Lstat(path)
	mist(t, err)
	return stat.Mode()
}

func putfile(t *testing.T, basePath string, i int, data []byte) {
	putfileEx(t, basePath, i, data, os.FileMode(0777))
}

func putfileEx(t *testing.T, basePath string, i int, data []byte, perm os.FileMode) {
	samplePath := path.Join(basePath, fmt.Sprintf("dummy%d.dat", i))
	mist(t, ioutil.WriteFile(samplePath, data, perm))
}

func TestAllTheThings(t *testing.T) {
	perm := os.FileMode(0777)
	workingDir, err := ioutil.TempDir("", "butler-tests")
	mist(t, err)
	defer os.RemoveAll(workingDir)

	sample := path.Join(workingDir, "sample")
	mist(t, os.MkdirAll(sample, perm))
	mist(t, ioutil.WriteFile(path.Join(sample, "hello.txt"), []byte("hello!"), perm))

	sample2 := path.Join(workingDir, "sample2")
	mist(t, os.MkdirAll(sample2, perm))
	for i := 0; i < 5; i++ {
		if i == 3 {
			// e.g. .gitkeep
			putfile(t, sample2, i, []byte{})
		} else {
			putfile(t, sample2, i, bytes.Repeat([]byte{0x42, 0x69}, i*200+1))
		}
	}

	sample3 := path.Join(workingDir, "sample3")
	mist(t, os.MkdirAll(sample3, perm))
	for i := 0; i < 60; i++ {
		putfile(t, sample3, i, bytes.Repeat([]byte{0x42, 0x69}, i*300+1))
	}

	sample4 := path.Join(workingDir, "sample4")
	mist(t, os.MkdirAll(sample4, perm))
	for i := 0; i < 120; i++ {
		putfile(t, sample4, i, bytes.Repeat([]byte{0x42, 0x69}, i*150+1))
	}

	sample5 := path.Join(workingDir, "sample5")
	mist(t, os.MkdirAll(sample5, perm))
	rg := rand.New(rand.NewSource(0x239487))

	for i := 0; i < 25; i++ {
		l := 1024 * (i + 2)
		// our own little twist on fizzbuzz to look out for 1-off errors
		if i%5 == 0 {
			l = int(pwr.BlockSize)
		} else if i%3 == 0 {
			l = 0
		}

		buf := make([]byte, l)
		_, err := io.CopyN(bytes.NewBuffer(buf), rg, int64(l))
		mist(t, err)
		putfile(t, sample5, i, buf)
	}

	files := map[string]string{
		"hello":     sample,
		"80-fixed":  sample2,
		"60-fixed":  sample3,
		"120-fixed": sample4,
		"random":    sample5,
		"null":      "/dev/null",
	}

	patch := path.Join(workingDir, "patch.pwr")

	comm.Configure(true, true, false, false, false, false, false)

	for _, q := range []int{1, 9} {
		t.Logf("============ Quality %d ============", q)
		compression := pwr.CompressionSettings{
			Algorithm: pwr.CompressionAlgorithm_BROTLI,
			Quality:   int32(q),
		}

		for lhs := range files {
			for rhs := range files {
				mist(t, diff.Do(&diff.Params{
					Target:      files[lhs],
					Source:      files[rhs],
					Patch:       patch,
					Compression: compression,
				}))
				stat, err := os.Lstat(patch)
				mist(t, err)
				t.Logf("%10s -> %10s = %s", lhs, rhs, humanize.IBytes(uint64(stat.Size())))
			}
		}
	}

	compression := pwr.CompressionSettings{
		Algorithm: pwr.CompressionAlgorithm_BROTLI,
		Quality:   1,
	}

	for _, filepath := range files {
		t.Logf("Signing %s\n", filepath)

		sigPath := path.Join(workingDir, "signature.pwr.sig")
		mist(t, sign.Do(filepath, sigPath, compression, false))

		sigReader, err := eos.Open(sigPath)
		mist(t, err)

		sigSource := seeksource.FromFile(sigReader)
		_, err = sigSource.Resume(nil)
		mist(t, err)

		signature, err := pwr.ReadSignature(sigSource)
		mist(t, err)

		mist(t, sigReader.Close())

		validator := &pwr.ValidatorContext{
			FailFast: true,
		}

		mist(t, validator.Validate(filepath, signature))
	}

	// K windows you just sit this one out we'll catch you on the flip side
	if runtime.GOOS != "windows" {
		// In-place preserve permissions tests
		t.Logf("In-place patching should preserve permissions")

		eperm := os.FileMode(0750)

		samplePerm1 := path.Join(workingDir, "samplePerm1")
		mist(t, os.MkdirAll(samplePerm1, perm))
		putfileEx(t, samplePerm1, 1, bytes.Repeat([]byte{0x42, 0x69}, 8192), eperm)

		assert.Equal(t, octal(eperm), octal(permFor(t, path.Join(samplePerm1, "dummy1.dat"))))

		samplePerm2 := path.Join(workingDir, "samplePerm2")
		mist(t, os.MkdirAll(samplePerm2, perm))
		putfileEx(t, samplePerm2, 1, bytes.Repeat([]byte{0x69, 0x42}, 16384), eperm)

		assert.Equal(t, octal(eperm), octal(permFor(t, path.Join(samplePerm2, "dummy1.dat"))))

		mist(t, diff.Do(&diff.Params{
			Target:      samplePerm1,
			Source:      samplePerm2,
			Patch:       patch,
			Compression: compression,
		}))
		_, err := os.Lstat(patch)
		mist(t, err)

		cave := path.Join(workingDir, "cave")
		ditto.Do(samplePerm1, cave)

		mist(t, apply.Do(&apply.Params{
			Patch:   patch,
			Target:  cave,
			Output:  cave,
			InPlace: true,
		}))
		assert.Equal(t, octal(eperm|pwr.ModeMask), octal(permFor(t, path.Join(cave, "dummy1.dat"))))
	}
}
