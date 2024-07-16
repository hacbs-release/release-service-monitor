/*
Copyright 2024 Red Hat Inc

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package checks

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"

	"github.com/containers/common/libimage"
	"github.com/containers/common/pkg/config"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/types"

	"github.com/hacbs-release/release-availability-metrics/pkg/metrics"
)

// A QuayCheck sets the necessary parameters to run a check to quay.io.
type QuayCheck struct {
	name   string
	ctx    context.Context
	auth   QuayAuth
	image  string
	tags   []string
	log    *log.Logger
	metric metrics.GaugeMetric
}

// MewQuayCheck creates a new QuayCheck instance.
func NewQuayCheck(ctx context.Context, auth *QuayAuth, name string, image string, tags []string, log *log.Logger,
	metric metrics.GaugeMetric) *QuayCheck {
	log.Println("creating new Quay check")
	newCheck := &QuayCheck{
		name:   name,
		ctx:    ctx,
		auth:   *auth,
		image:  image,
		tags:   tags,
		log:    log,
		metric: metric,
	}

	return newCheck
}

// pullImage pulls a image from quay and returns CheckResult and nil in case of success or CheckResult and error
// in case of failure.
func (c *QuayCheck) pullImage() (CheckResult, error) {

	var (
		runtimeOptions libimage.RuntimeOptions
		runtime        *libimage.Runtime
		store          storage.Store
		storeOptions   types.StoreOptions
		tmpdir         string
		err            error
	)

	c.log.Println(fmt.Sprintf("fetching image %s as %s", c.getImage(), c.auth.getUsername()))

	storeOptions, err = types.DefaultStoreOptions()
	if err != nil {
		c.log.Println(fmt.Sprintf("check failed: %s", err.Error()))
		return CheckResult{1, "Failed", err.Error()}, err
	}

	tmpdir, err = os.MkdirTemp("/tmp", "quaycheck-")
	c.log.Println(fmt.Sprintf("temporary directory is %s", tmpdir))
	if err != nil {
		c.log.Println(fmt.Sprintf("check failed: %s", err.Error()))
		return CheckResult{1, "Failed", err.Error()}, err
	}

	storageDir := fmt.Sprintf("%s/%d/", tmpdir, os.Getuid())
	storeOptions.RunRoot = storageDir
	storeOptions.GraphRoot = storageDir
	storeOptions.RootlessStoragePath = storageDir
	storeOptions.GraphDriverName = "vfs"
	storeOptions.GraphDriverOptions = []string{"vfs.ignore_chown_errors=1"}
	storeOptions.RootAutoNsUser = string(os.Getuid())

	storeOptions.UIDMap = []idtools.IDMap{{
		ContainerID: 0,
		HostID:      os.Getuid(),
		Size:        1,
	}}

	storeOptions.GIDMap = []idtools.IDMap{{
		ContainerID: 0,
		HostID:      os.Getgid(),
		Size:        1,
	}}

	store, err = storage.GetStore(storeOptions)
	if err != nil {
		c.log.Println(fmt.Sprintf("check failed: %s", err.Error()))
		return CheckResult{1, "Failed", err.Error()}, err
	}

	runtime, err = libimage.RuntimeFromStore(store, &runtimeOptions)
	if err != nil {
		c.log.Println(fmt.Sprintf("check failed: %s", err.Error()))
		return CheckResult{1, "Failed", err.Error()}, err
	}

	options := &libimage.PullOptions{}
	options.Username = c.auth.getUsername()
	options.Password = c.auth.getPassword()
	options.Writer = io.Discard

	pullImage := ""
	for i := 0; i < len(c.tags); i++ {
		// pull the image with tag unless no tag is set
		pullImage = fmt.Sprintf("%s:%s", c.getImage(), c.tags[i])
		if c.tags[i] == "" {
			pullImage = fmt.Sprintf("%s", c.getImage())
		}

		_, err = runtime.Pull(c.ctx, pullImage, config.PullPolicyAlways, options)
		if err != nil {
			// mount error is expected but we don't need the mounting to
			// assure the image was reacheable. A error message is also
			// displayed in the console, but can be ignored.
			re := regexp.MustCompile(`.*creating mount namespace.*`)
			if re.FindString(err.Error()) != "" {
				// check next image
				continue
			}

			c.log.Println(fmt.Sprintf("check failed: %s", err.Error()))
			return CheckResult{1, "Failed", err.Error()}, err
		}
	}

	c.log.Println(c.name, "check succeeded")
	return CheckResult{0, "Succeeded", ""}, nil
}

// Check runs a QuayCheck and return the float64 status required to save the prometheus data.
func (c *QuayCheck) Check() float64 {
	var reason string

	c.log.Println("running quay check:", c.name)
	pull, err := c.pullImage()
	if err != nil {
		reason = err.Error()
	}
	c.metric.Record([]string{c.name, reason, pull.status}, metrics.FlipValue(pull.code))

	return pull.code
}

// getImage returns the image parameter of a QuayCheck instance.
func (c *QuayCheck) getImage() string {
	return c.image
}
