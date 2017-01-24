package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
)

// Used for Plugin discovery.
// Docker identifies the existence of an active plugin process by seraching for
// a unit socket file (.sock) file in /run/docker/plugins/.
// A unix server is started at the `socketAdress` to enable discovery of this plugin by docker.
const (
	socketAddress = "/run/docker/plugins/minfs.sock"
)

// configuration values of the remote Minio server.
// Minfs uses this info the mount the remote bucket.
// The server info (endpoint, accessKey and secret Key) is passed during creating a docker volume.
// Here is how to do it,
// $ docker volume create -d minfs-plugin \
//    --name medical-imaging-store \
//     -o endpoint=https://play.minio.io:9000/rao -o access_key=Q3AM3UQ867SPQQA43P2F\
//     -o secret-key=zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG -o mountpoint=/minfs
//
type serverConfig struct {
	// Endpoint of the remote Minio server.
	endpoint string
	// `minfs` mounts the remote bucket to a the local `mountpoint`.
	bucket string
	// accessKey of the remote minio server.
	accessKey string
	// secretKey of the remote Minio server.
	secretKey string
}

// represents an instance of `minfs` mount of remote Minio bucket.
// Its defined by
//   - The server info for the mount.
//   - The local mountpoint.
//   - The number of connections alive for the mount (No.Of.Services still using the mount point).
type mountInfo struct {
	serverconfig serverConfig
	mountPoint   string
	// the number of containers using the mount.
	// an active mount is done when the count is 0.
	// unmount is done only if the number of connections is 0.
	connections int
}

// minfsDriver - The struct implements the `github.com/docker/go-plugins-helpers/volume.Driver` interface.
// Here are the sequence of events that defines the interaction between docker and the plugin server.
// 1. Implement the interface defined in `github.com/docker/go-plugins-helpers/volume.Driver`.
//    In our case the struct `minfsDriver` implements the interface.
// 2. Create a new instance of `minfsDriver` and register it with the `go-plugin-helper`.
//    `go-plugin-helper` is a tool built to make development of docker plugins easier, visit https://github.com/docker/go-plugins-helpers/.
//     The registration is done using https://godoc.org/github.com/docker/go-plugins-helpers/volume#NewHandler .
// 3. Docker interacts with the plugin server via HTTP endpoints whose
//    protocols defined here https://docs.docker.com/engine/extend/plugins_volume/#/volumedrivercreate.
// 4. Once registered the implemented methods on `minfsDriver` are called whenever docker
//    interacts with the plugin via HTTP requests. These methods are resposible for responding to docker with
//    success or error messages.
type minfsDriver struct {
	// used for atomic access to the fields.
	sync.RWMutex
	mountRoot string
	// config of the remote Minio server.
	config serverconfig
	// the local path to which the remote Minio bucket is mounted to.

	// An active volume driver server can be used to mount multiple
	// remote buckets possibly even referring to even different Minio server
	// instances or buckets.
	// The state info of these mounts are maintained here.
	mounts map[string]*mountInfo
}

// return a new instance of minfsDriver.
func newMinfsDriver(mountRoot string) *minfsDriver {
	logrus.WithField("method", "new minfs driver").Debug(root)

	d := &minfsDriver{
		mountRoot: mountRoot,
		config:    serverConfig,
		mounts:    make(map[string]*mountInfo),
	}

	return d
}

// *minfsDriver.Create - This method is called by docker when a volume is created
//                       using `$docker volume create -d <plugin-name> --name <volume-name>`.
// the name (--name) of the plugin uniquely identifies the mount.
// The name of the plugin is passed by docker to the plugin during the HTTP call, check
// https://docs.docker.com/engine/extend/plugins_volume/#/volumedrivercreate for more details.
// Additional options can be passed only during call to `Create`,
// $ docker volume create -d <plugin-name> --name <volume-name> -o <option-key>=<option-value>
// The name of the volume uniquely identifies the mount.
// The remote bucket will be mounted at `mountRoot + volumeName`.
// mountRoot is passed as `--mountroot` flag when starting the server.
func (d *minfsDriver) Create(r volume.Request) volume.Response {
	logrus.WithField("method", "Create").Debugf("%#v", r)
	// hold lock for safe access.
	d.Lock()
	defer d.Unlock()
	// validate the inputs.
	// verify that the name of the volume is not empty.
	if r.Name == "" {
		return errorResponse("Name of the driver cannot be empty.Use `$ docker volume create -d <plugin-name> --name <volume-name>`")
	}
	// TODO: verify whether a volume by the given name already exists.
	// if the volume is already created verify that the server configs match.
	// If not return with error/
	if ok := d.mounts[r.Name]; ok {

	}

	// TODO: Verify if the bucket by the name of the volume exists.
	// If it doesnt exist create the bucket on the remote Minio server.

	// verify that all the options are set when the volume is created.
	if r.Options == nil {
		return errorResponse("No options provided. Please refer example usage.")
	}
	if r.Options["endpoint"] == "" {
		return errorResponse("endpoint option cannot be empty.")
	}
	if r.Options["bucket"] == "" {
		return errorResponse("bucket option cannot be empty.")
	}
	if r.Options["access-key"] == "" {
		return errorResponse("access-key option cannot be empty")
	}
	if r.Options["secret-key"] == "" {
		return errorResponse("secret-key cannot be empty.")
	}

	mntInfo := &mountInfo{}
	config := serverConfig{}

	// Additional options passed with `-o` option are parsed here.
	config.endpoint = r.Options["endpoint"]
	config.bucket = r.Options["bucket"]
	config.secretKey = r.Options["secret-key"]
	config.accessKey = r.Options["access-key"]

	mntInfo.mountPoint = filepath.Join(d.mountRoot, r.Name)
	mntInfo.Config = config
	// `r.Name` contains the plugin name passed with `--name` in `$ docker volume create -d <plugin-name> --name <volume-name>`.
	// Name of the volume uniquely identiifies the mount.
	d.volumes[r.Name] = v
	return volume.Response{}
}

// Error repsonse to be sent to docker on failure of any operation.
func errorResponse(err string) volume.Response {
	logrus.Error(err)
	return volume.Response{Err: err}
}

// TODO : Add comments, clean up and fix errors.
func (d *minfsDriver) Remove(r volume.Request) volume.Response {
	logrus.WithField("method", "remove").Debugf("%#v", r)

	d.Lock()
	defer d.Unlock()

	v, ok := d.volumes[r.Name]
	if !ok {
		return responseError(fmt.Sprintf("volume %s not found", r.Name))
	}

	if v.connections == 0 {
		if err := os.RemoveAll(v.mountpoint); err != nil {
			return responseError(err.Error())
		}
		delete(d.volumes, r.Name)
		return volume.Response{}
	}
	return responseError(fmt.Sprintf("volume %s is currently used by a container", r.Name))
}

func (d *minfsDriver) Path(r volume.Request) volume.Response {
	logrus.WithField("method", "path").Debugf("%#v", r)

	d.RLock()
	defer d.RUnlock()

	v, ok := d.volumes[r.Name]
	if !ok {
		return responseError(fmt.Sprintf("volume %s not found", r.Name))
	}

	return volume.Response{Mountpoint: v.mountpoint}
}

func (d *minfsDriver) Mount(r volume.MountRequest) volume.Response {
	logrus.WithField("method", "mount").Debugf("%#v", r)

	d.Lock()
	defer d.Unlock()

	v, ok := d.volumes[r.Name]
	if !ok {
		return responseError(fmt.Sprintf("volume %s not found", r.Name))
	}

	if v.connections > 0 {
		v.connections++
		return volume.Response{Mountpoint: v.mountpoint}
	}

	fi, err := os.Lstat(v.mountpoint)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(v.mountpoint, 0755); err != nil {
			return responseError(err.Error())
		}
	} else if err != nil {
		return responseError(err.Error())
	}

	if fi != nil && !fi.IsDir() {
		return responseError(fmt.Sprintf("%v already exist and it's not a directory", v.mountpoint))
	}

	if err := d.mountVolume(v); err != nil {
		return responseError(err.Error())
	}

	return volume.Response{Mountpoint: v.mountpoint}
}

func (d *minfsDriver) Unmount(r volume.UnmountRequest) volume.Response {
	logrus.WithField("method", "unmount").Debugf("%#v", r)

	d.Lock()
	defer d.Unlock()
	v, ok := d.volumes[r.Name]
	if !ok {
		return responseError(fmt.Sprintf("volume %s not found", r.Name))
	}
	if v.connections <= 1 {
		if err := d.unmountVolume(v.mountpoint); err != nil {
			return responseError(err.Error())
		}
		v.connections = 0
	} else {
		v.connections--
	}

	return volume.Response{}
}

func (d *minfsDriver) Get(r volume.Request) volume.Response {
	logrus.WithField("method", "get").Debugf("%#v", r)

	d.Lock()
	defer d.Unlock()

	v, ok := d.volumes[r.Name]
	if !ok {
		return responseError(fmt.Sprintf("volume %s not found", r.Name))
	}

	return volume.Response{Volume: &volume.Volume{Name: r.Name, Mountpoint: v.mountpoint}}
}

func (d *minfsDriver) List(r volume.Request) volume.Response {
	logrus.WithField("method", "list").Debugf("%#v", r)

	d.Lock()
	defer d.Unlock()

	var vols []*volume.Volume
	for name, v := range d.volumes {
		vols = append(vols, &volume.Volume{Name: name, Mountpoint: v.mountpoint})
	}
	return volume.Response{Volumes: vols}
}

func (d *minfsDriver) Capabilities(r volume.Request) volume.Response {
	logrus.WithField("method", "capabilities").Debugf("%#v", r)

	return volume.Response{Capabilities: volume.Capability{Scope: "local"}}
}

func (d *minfsDriver) mountVolume(v *mountInfo) error {
	// TODO: mount here.
	cmd := fmt.Sprintf("<mount here>")

	logrus.Debug(cmd)
	return exec.Command("sh", "-c", cmd).Run()
}

func (d *minfsDriver) unmountVolume(target string) error {
	// TODO: Unmount here.
	cmd := fmt.Sprintf("umount %s", target)
	logrus.Debug(cmd)
	return exec.Command("sh", "-c", cmd).Run()
}

func main() {
	mountRoot := flag.String("mountroot", "/tmp", "root for mouting Minio buckets.")
	// check if the mount root exists.
	// create if it doesn't exist.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(*mountRoot, 0600)
	} else {
		logrus.Error(err)
	}

	debug := os.Getenv("DEBUG")
	if ok, _ := strconv.ParseBool(debug); ok {
		logrus.SetLevel(logrus.DebugLevel)
	}

	d := newMinfsDriver(*mountRoot)
	h := volume.NewHandler(d)
	logrus.Infof("listening on %s", socketAddress)
	logrus.Error(h.ServeUnix(socketAddress, 0))
}
