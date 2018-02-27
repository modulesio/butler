package prereqs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/modulesio/isolator/manager"

	"github.com/go-errors/errors"
	"github.com/modulesio/isolator/buse"
	"github.com/modulesio/isolator/cmd/elevate"
	"github.com/modulesio/isolator/cmd/operate"
	"github.com/modulesio/isolator/installer"
	"github.com/mitchellh/mapstructure"
)

func (pc *PrereqsContext) InstallPrereqs(tsc *TaskStateConsumer, plan *PrereqPlan) error {
	consumer := pc.Consumer

	needElevation := false
	for _, task := range plan.Tasks {
		switch pc.Runtime.Platform {
		case manager.ItchPlatformWindows:
			block := task.Info.Windows
			if block.Elevate {
				consumer.Infof("Will perform prereqs installation elevated because of (%s)", task.Name)
				needElevation = true
			}
		case manager.ItchPlatformLinux:
			block := task.Info.Linux
			if len(block.EnsureSuidRoot) > 0 {
				consumer.Infof("Will perform prereqs installation elevated because (%s) has SUID binaries", task.Name)
				needElevation = true
			}
		}
	}

	planFile, err := ioutil.TempFile("", "butler-prereqs-plan.json")
	if err != nil {
		return errors.Wrap(err, 0)
	}

	planPath := planFile.Name()
	defer os.Remove(planPath)

	enc := json.NewEncoder(planFile)
	err = enc.Encode(plan)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = planFile.Close()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	var args []string
	if needElevation {
		args = append(args, "--elevate")
	}
	args = append(args, []string{"install-prereqs", planPath}...)

	res, err := installer.RunSelf(&installer.RunSelfParams{
		Consumer: consumer,
		Args:     args,
		OnResult: func(value installer.Any) {
			switch value["type"] {
			case "state":
				{
					ps := &PrereqState{}
					msdec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
						TagName: "json",
						Result:  ps,
					})
					if err != nil {
						consumer.Warnf("could not decode result: %s", err.Error())
						return
					}

					err = msdec.Decode(value)
					if err != nil {
						consumer.Warnf("could not decode result: %s", err.Error())
						return
					}

					tsc.OnState(&buse.PrereqsTaskStateNotification{
						Name:   ps.Name,
						Status: ps.Status,
					})
				}
			}
		},
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if res.ExitCode != 0 {
		if res.ExitCode == elevate.ExitCodeAccessDenied {
			return operate.ErrAborted
		}
	}

	err = installer.CheckExitCode(res.ExitCode, err)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	// now to run some sanity checks (as regular user)
	for _, task := range plan.Tasks {
		switch pc.Runtime.Platform {
		case manager.ItchPlatformLinux:
			block := task.Info.Linux
			for _, sc := range block.SanityChecks {
				err := pc.RunSanityCheck(task.Name, task.Info, sc)
				if err != nil {
					retErr := fmt.Errorf("Sanity check failed for (%s): %s", task.Name, err.Error())
					return errors.Wrap(retErr, 0)
				}
				consumer.Infof("Sanity check (%s ::: %s) passed", sc.Command, strings.Join(sc.Args, " ::: "))
			}
		}
	}

	return nil
}
