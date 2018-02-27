package push

import (
	"fmt"
	"io/ioutil"
	"math"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/modulesio/isolator/comm"
	"github.com/modulesio/isolator/mansion"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/httpkit/uploader"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
)

const (
	// almostThereThreshold is the amount of data left where the progress indicator isn't indicative anymore.
	// At this point, we're basically waiting for build files to be finalized.
	almostThereThreshold int64 = 10 * 1024
)

var args = struct {
	src             *string
	target          *string
	userVersion     *string
	userVersionFile *string
	fixPerms        *bool
	dereference     *bool
	ifChanged       *bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("push", "Upload a new build to itch.io. See `butler help push`.")
	args.src = cmd.Arg("src", "Directory to upload. May also be a zip archive (slower)").Required().String()
	args.target = cmd.Arg("target", "Where to push, for example 'leafo/x-moon:win-64'. Targets are of the form project:channel, where project is username/game or game_id.").Required().String()
	args.userVersion = cmd.Flag("userversion", "A user-supplied version number that you can later query builds by").String()
	args.userVersionFile = cmd.Flag("userversion-file", "A file containing a user-supplied version number that you can later query builds by").String()
	args.fixPerms = cmd.Flag("fix-permissions", "Detect Mac & Linux executables and adjust their permissions automatically").Default("true").Bool()
	args.dereference = cmd.Flag("dereference", "Dereference symlinks").Default("false").Bool()
	args.ifChanged = cmd.Flag("if-changed", "Don't push anything if it would be an empty patch").Default("false").Bool()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	go ctx.DoVersionCheck()

	// if userVersionFile specified, read from the given file
	// TODO: do utf-16 decoding here
	userVersion := *args.userVersion
	if userVersion == "" && *args.userVersionFile != "" {
		buf, err := ioutil.ReadFile(*args.userVersionFile)
		ctx.Must(err)

		userVersion = strings.TrimSpace(string(buf))
		if strings.ContainsAny(userVersion, "\r\n") {
			ctx.Must(fmt.Errorf("%s contains line breaks, refusing to use as userversion", *args.userVersionFile))
		}
	}

	ctx.Must(Do(ctx, *args.src, *args.target, userVersion, *args.fixPerms, *args.dereference, *args.ifChanged))
}

func Do(ctx *mansion.Context, buildPath string, specStr string, userVersion string, fixPerms bool, dereference bool, ifChanged bool) error {
	// start walking source container while waiting on auth flow
	sourceContainerChan := make(chan walkResult)
	walkErrs := make(chan error)
	go doWalk(buildPath, sourceContainerChan, walkErrs, fixPerms, dereference)

	spec, err := itchio.ParseSpec(specStr)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = spec.EnsureChannel()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	client, err := ctx.AuthenticateViaOauth()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	getSignature := func(ID int64) (*pwr.SignatureInfo, error) {
		buildFiles, err := client.ListBuildFiles(ID)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		signatureFile := itchio.FindBuildFile(itchio.BuildFileTypeSignature, buildFiles.Files)
		if signatureFile == nil {
			comm.Dief("Could not find signature for parent build %d, aborting", ID)
		}

		signatureURL := itchio.ItchfsURL(
			ID,
			signatureFile.ID,
			client.Key,
		)

		signatureReader, err := eos.Open(signatureURL)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}
		defer signatureReader.Close()

		signatureSource := seeksource.FromFile(signatureReader)

		_, err = signatureSource.Resume(nil)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		signature, err := pwr.ReadSignature(signatureSource)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		return signature, nil
	}

	if ifChanged {
		chanInfo, err := client.GetChannel(spec.Target, spec.Channel)
		if err == nil && chanInfo != nil && chanInfo.Channel != nil && chanInfo.Channel.Head != nil {
			comm.Opf("Comparing against previous build...")
			sig, err := getSignature(chanInfo.Channel.Head.ID)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			err = pwr.AssertValid(buildPath, sig)
			if err == nil {
				comm.Statf("No changes and --if-changed used, not pushing anything")
				return nil
			}

			if _, ok := err.(*pwr.ErrHasWound); ok {
				// cool, that's what we expected
			} else {
				return errors.Wrap(err, 0)
			}
		} else {
			comm.Opf("No previous build to compare against, pushing unconditionally")
		}
	}

	newBuildRes, err := client.CreateBuild(spec.Target, spec.Channel, userVersion)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	buildID := newBuildRes.Build.ID
	parentID := newBuildRes.Build.ParentBuild.ID

	var targetSignature *pwr.SignatureInfo

	if parentID == 0 {
		comm.Opf("For channel `%s`: pushing first build", spec.Channel)
		targetSignature = &pwr.SignatureInfo{
			Container: &tlc.Container{},
			Hashes:    make([]wsync.BlockHash, 0),
		}
	} else {
		comm.Opf("For channel `%s`: last build is %d, downloading its signature", spec.Channel, parentID)
		var err error
		targetSignature, err = getSignature(parentID)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	newPatchRes, newSignatureRes, err := createBothFiles(client, buildID)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	consumer := comm.NewStateConsumer()

	patchWriter := uploader.NewResumableUpload2(newPatchRes.File.UploadURL)
	patchWriter.SetConsumer(consumer)

	signatureWriter := uploader.NewResumableUpload2(newSignatureRes.File.UploadURL)
	signatureWriter.SetConsumer(consumer)

	comm.Debugf("Launching patch & signature channels")

	patchCounter := counter.NewWriter(patchWriter)
	signatureCounter := counter.NewWriter(signatureWriter)

	// we started walking the source container in the beginning,
	// we actually need it now.
	// note that we could actually start diffing before all the file
	// creation & upload setup is done

	var sourceContainer *tlc.Container
	var sourcePool wsync.Pool

	comm.Debugf("Waiting for source container")
	select {
	case walkErr := <-walkErrs:
		return errors.Wrap(walkErr, 1)
	case walkies := <-sourceContainerChan:
		comm.Debugf("Got sourceContainer!")
		sourceContainer = walkies.container
		sourcePool = walkies.pool
		break
	}

	if sourceContainer.IsSingleFile() {
		comm.Notice("You're pushing a single file", []string{
			"Diffing and patching work poorly on 'all-in-one executables' and installers. Consider pushing a portable build instead, for optimal distribution.",
			"",
			"For more information, see https://itch.io/docs/butler/single-files.html",
		})
	}

	comm.Opf("Pushing %s (%s)", humanize.IBytes(uint64(sourceContainer.Size)), sourceContainer.Stats())

	comm.Debugf("Building diff context")
	var readBytes int64

	var bytesPerSec float64
	var lastUploadedBytes int64
	var patchUploadedBytes int64

	stopTicking := make(chan struct{})
	updateProgress := func() {
		// input bytes that aren't in output, for example:
		//  - bytes that have been compressed away
		//  - bytes that were in old build and were simply reused
		goneBytes := readBytes - patchCounter.Count()

		conservativeTotalBytes := sourceContainer.Size - goneBytes

		leftBytes := conservativeTotalBytes - patchUploadedBytes
		if leftBytes > almostThereThreshold {
			netStatus := "- network idle"
			if bytesPerSec > 1 {
				netStatus = fmt.Sprintf("@ %s/s", humanize.IBytes(uint64(bytesPerSec)))
			}
			comm.ProgressLabel(fmt.Sprintf("%s, %s left", netStatus, humanize.IBytes(uint64(leftBytes))))
		} else {
			comm.ProgressLabel(fmt.Sprintf("- almost there"))
		}

		conservativeProgress := float64(patchUploadedBytes) / float64(conservativeTotalBytes)
		conservativeProgress = min(1.0, conservativeProgress)
		comm.Progress(conservativeProgress)

		comm.ProgressScale(float64(readBytes) / float64(sourceContainer.Size))
	}

	patchWriter.SetProgressListener(func(count int64) {
		patchUploadedBytes = count
		updateProgress()
	})

	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(2))
		for {
			select {
			case <-ticker.C:
				bytesPerSec = float64(patchUploadedBytes-lastUploadedBytes) / 2.0
				lastUploadedBytes = patchUploadedBytes
				updateProgress()
			case <-stopTicking:
				return
			}
		}
	}()

	stateConsumer := &state.Consumer{
		OnProgress: func(progress float64) {
			readBytes = int64(float64(sourceContainer.Size) * progress)
			updateProgress()
		},
	}

	dctx := &pwr.DiffContext{
		Compression: &pwr.CompressionSettings{
			Algorithm: pwr.CompressionAlgorithm_BROTLI,
			Quality:   1,
		},

		SourceContainer: sourceContainer,
		Pool:            sourcePool,

		TargetContainer: targetSignature.Container,
		TargetSignature: targetSignature.Hashes,

		Consumer: stateConsumer,
	}

	comm.StartProgress()
	comm.ProgressScale(0.0)
	err = dctx.WritePatch(patchCounter, signatureCounter)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	// close both files concurrently
	{
		errs := make(chan error)

		go func() {
			errs <- patchWriter.Close()
		}()
		go func() {
			errs <- signatureWriter.Close()
		}()

		// 2 close
		for i := 0; i < 2; i++ {
			err := <-errs
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
	}

	close(stopTicking)
	comm.ProgressLabel("finalizing build")

	// finalize both files concurrently
	{
		errs := make(chan error)

		doFinalize := func(fileID int64, fileSize int64, done chan error) {
			_, err = client.FinalizeBuildFile(buildID, fileID, fileSize)
			done <- err
		}

		go doFinalize(newPatchRes.File.ID, patchCounter.Count(), errs)
		go doFinalize(newSignatureRes.File.ID, signatureCounter.Count(), errs)

		// 2 doFinalize
		for i := 0; i < 2; i++ {
			err := <-errs
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
	}

	comm.EndProgress()

	{
		prettyPatchSize := humanize.IBytes(uint64(patchCounter.Count()))
		percReused := 100.0 * float64(dctx.ReusedBytes) / float64(dctx.FreshBytes+dctx.ReusedBytes)
		relToNew := 100.0 * float64(patchCounter.Count()) / float64(sourceContainer.Size)
		prettyFreshSize := humanize.IBytes(uint64(dctx.FreshBytes))
		savings := 100.0 - relToNew

		if dctx.ReusedBytes > 0 {
			comm.Statf("Re-used %.2f%% of old, added %s fresh data", percReused, prettyFreshSize)
		} else {
			comm.Statf("Added %s fresh data", prettyFreshSize)
		}

		if savings > 0 && !math.IsNaN(savings) {
			comm.Statf("%s patch (%.2f%% savings)", prettyPatchSize, 100.0-relToNew)
		} else {
			comm.Statf("%s patch (no savings)", prettyPatchSize)
		}
	}
	comm.Opf("Build is now processing, should be up in a bit (see `butler status`)")
	comm.Logf("")

	return nil
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
