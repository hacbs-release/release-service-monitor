package checks

import (

    "io"
    "os"
    "fmt"
    "regexp"
    "context"

    "github.com/containers/common/libimage"
    "github.com/containers/common/pkg/config"
    "github.com/containers/storage"
    "github.com/containers/storage/types"
    "github.com/containers/storage/pkg/idtools"

    "github.com/hacbs-release/release-availability-metrics/pkg/metrics"
)

// A quayCheck sets the necessary parameters to run a check to quay.io.
type quayCheck struct {
    name  string
    ctx   context.Context
    auth  QuayAuth
    image string
    tags  []string
}

// MewQuayCheck creates a new quayCheck instance.
func NewQuayCheck(ctx context.Context, auth *QuayAuth, image string, tags []string) (*quayCheck) {
    newCheck := &quayCheck{
        name:  "quay",
        ctx:   ctx,
        auth:  *auth,
        image: image,
        tags:  tags,
    }

    return newCheck
}

// pullImage pulls a image from quay and returns CheckResult and nil in case of success or CheckResult and error
// in case of failure.
func (c *quayCheck) pullImage() (CheckResult, error) {

    var (
        runtimeOptions libimage.RuntimeOptions
        runtime        *libimage.Runtime
        store          storage.Store
        storeOptions   types.StoreOptions
        err            error
    )

    storeOptions, err = types.DefaultStoreOptions()
    if err != nil {
        return CheckResult{1, "Failed", err.Error()}, err
    }

    storageDir := fmt.Sprintf("%s/%d/containers/storage", os.Getenv("HOME"), os.Getuid())

    storeOptions.RunRoot = storageDir
    storeOptions.GraphRoot = storageDir
    storeOptions.RootlessStoragePath = storageDir
    storeOptions.GraphDriverName = "vfs"
    storeOptions.GraphDriverOptions = []string{"vfs.ignore_chown_errors=1"}
    storeOptions.RootAutoNsUser = string(os.Getuid())

    storeOptions.UIDMap = []idtools.IDMap{{
        ContainerID: 0,
        HostID: os.Getuid(),
        Size: 1,
    }}

    storeOptions.GIDMap = []idtools.IDMap{{
        ContainerID: 0,
        HostID: os.Getgid(),
        Size: 1,
    }}

    store, err = storage.GetStore(storeOptions)
    if err != nil {
        return CheckResult{1, "Failed", err.Error()}, err
    }

    runtime, err = libimage.RuntimeFromStore(store, &runtimeOptions)
    if err != nil {
        return CheckResult{1, "Failed", err.Error()}, err
    }

    options := &libimage.PullOptions{}
    options.Username = c.auth.getUsername()
    options.Password = c.auth.getPassword()
    options.Writer = io.Discard
    
    pullImage := ""
    for i := 0; i< len(c.tags); i++ {
        // pull the image with tag unless no tag is set
        pullImage = fmt.Sprintf("%s:%s", c.getImage(), c.tags[i])
        if c.tags[i] == "" {
            pullImage = fmt.Sprintf("%s", c.getImage())
        }

        _, err = runtime.Pull(c.ctx, pullImage, config.PullPolicyAlways, options)
        if err != nil {
            // mount error is expected but we don't need the mounting to
            // assure the image was reacheable
            re := regexp.MustCompile(`.*creating mount namespace.*`)
            if re.FindString(err.Error()) != "" {
                // check next image
                continue
            }

            return CheckResult{1, "Failed", err.Error()}, err
        }
    }

    return CheckResult{0, "Succeeded", ""}, nil
}

// Check runs a quayCheck and return the float64 status required to save the prometheus data
func (c *quayCheck) Check() (float64) {
    var reason string

    pull, err := c.pullImage()
    if err != nil {
        reason = err.Error()
    }

    metrics.RecordAvailabilityData(c.name, reason, pull.status, pull.code)

    return pull.code
}

// getImage returns the image parameter of a quayCheck instance
func (c *quayCheck) getImage() (string) {
    return c.image
}
