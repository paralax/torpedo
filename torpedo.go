package main

import (
	"fmt"
	"log"
	"os"

	"github.com/portworx/torpedo/scheduler"
)

// testDriverFunc runs a specific external storage test case.  It takes
// in a scheduler driver and an external volume provider (string) as
// arguments.
type testDriverFunc func(scheduler.Driver, string) error

const (
	// Use the inline volume specification so that we can test
	// volume options being dynamically parsed and used inline.
	volName = "size=10G,name=torpedo_fiovol"
)

// Create dynamic volumes.  Make sure that a task can use the dynamic volume
// in th einline format as size=x,repl=x,compress=x,name=foo.
func testDynamicVolume(
	d scheduler.Driver,
	volumeDriver string,
) error {
	// If it exists, remove it.
	d.RemoveVolume(volName)

	t := scheduler.Task{
		Name: "testDynamicVolume",
		Img:  "gourao/fio",
		Tag:  "latest",
		Cmd: []string{
			"fio",
			"--blocksize=64k",
			"--directory=/mnt/",
			"--ioengine=libaio",
			"--readwrite=write",
			"--size=5G",
			"--name=test",
			"--verify=meta",
			"--do_verify=1",
			"--verify_pattern=0xDeadBeef",
			"--direct=1",
			"--gtod_reduce=1",
			"--iodepth=1",
			"--randrepeat=1",
		},
		Vol: scheduler.Volume{
			Driver: volumeDriver,
			Name:   volName,
			Path:   "/mnt/",
			Size:   10240,
		},
	}

	if ctx, err := d.Create(t); err != nil {
		return err
	} else {
		defer func() {
			d.Destroy(ctx)
			d.RemoveVolume(volName)
		}()

		// Run the task and wait for completion.  This task will exit and
		// must not be re-started by the scheduler.
		if err = d.Run(ctx); err != nil {
			return err
		}

		if ctx.Status != 0 {
			return fmt.Errorf("Exit status %v\nStdout: %v\nStderr: %v\n",
				ctx.Status,
				ctx.Stdout,
				ctx.Stderr,
			)
		}
	}

	// Verify that the volume properties are honored.
	if v, err := d.InspectVolume(volName); err != nil {
		return err
	} else {
		// TODO: inspect size.
		/*
			if v.Size != 10240 {
				return fmt.Errorf(
					"Dynamic volume creation failed, size was not honored (size = %v).",
					v.Size,
				)
			}
		*/
		if v.Driver != volumeDriver {
			return fmt.Errorf(
				"Dynamic volume creation failed, incorrect volume driver (driver = %v).",
				v.Driver,
			)
		}
	}
	return nil
}

// Verify that the volume driver can deal with an uneven number of mounts
// and unmounts and allow the volume to get mounted on another node.
func testRemoteForceMount(
	d scheduler.Driver,
	volumeDriver string,
) error {
	return nil
}

// Volume Driver Plugin is down, unavailable - and the client container should
// not be impacted.
func testDriverDown(
	d scheduler.Driver,
	volumeDriver string,
) error {
	return nil
}

// Volume driver plugin is down and the client container gets terminated.
// There is a lost unmount call in this case, but the container should i
// be able to come up on another system and use the volume.
func testDriverDownContainerDown(
	d scheduler.Driver,
	volumeDriver string,
) error {
	return nil
}

// A container is using a volume on node X.  Node X is now powered off.
func testNodePowerOff(
	d scheduler.Driver,
	volumeDriver string,
) error {
	return nil
}

// Storage plugin is down.  Scheduler tries to create a container using the
// provider’s volume.
func testPluginDown(
	d scheduler.Driver,
	volumeDriver string,
) error {
	return nil
}

// A container is running on node X.  Node X loses network access and is
// partitioned away.  Node Y that is in the cluster can use the volume for
// another container.
func testNetworkDown(
	d scheduler.Driver,
	volumeDriver string,
) error {
	return nil
}

// A container is running on node X.  Node X can only see a subset of the
// storage cluster.  That is, it can see the entire DC/OS cluster, but just the
// storage cluster gets a network partition. Node Y that is in the cluster
// can use the volume for another container.
func testNetworkPartition(
	d scheduler.Driver,
	volumeDriver string,
) error {
	return nil
}

// Docker daemon crashes and live restore is disabled.
func testDockerDown(
	d scheduler.Driver,
	volumeDriver string,
) error {
	return nil
}

// Docker daemon crashes and live restore is enabled.
func testDockerDownLiveRestore(
	d scheduler.Driver,
	volumeDriver string,
) error {
	return nil
}

func run(d scheduler.Driver, vd string) error {
	if err := d.Init(); err != nil {
		return err
	}

	// Add new test functions here.
	testFuncs := map[string]testDriverFunc{
		"testDynamicVolume":           testDynamicVolume,
		"testRemoteForceMount":        testRemoteForceMount,
		"testDriverDown":              testDriverDown,
		"testDriverDownContainerDown": testDriverDownContainerDown,
		"testNodePowerOff":            testNodePowerOff,
		"testPluginDown":              testPluginDown,
		"testNetworkDown":             testNetworkDown,
		"testNetworkPartition":        testNetworkPartition,
		"testDockerDown":              testDockerDown,
		"testDockerDownLiveRestore":   testDockerDownLiveRestore,
	}

	for n, f := range testFuncs {
		// TODO: Delete test volumes before test, in case there was
		// previous state from a failed run.
		log.Printf("Executing test %v\n", n)
		if err := f(d, vd); err != nil {
			log.Printf("\tTest %v Failed with Error: %v.\n", n, err)
			return err
		}
		log.Printf("\tTest %v Passed.\n", n)
	}

	return nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: %v <scheduler> <volume driver>\n", os.Args[0])
		os.Exit(-1)
	}

	if d, err := scheduler.Get(os.Args[1]); err != nil {
		log.Fatalf("Cannot find driver %v\n", os.Args[1])
		os.Exit(-1)
	} else {
		if run(d, os.Args[2]) != nil {
			os.Exit(-1)
		}
	}

	log.Printf("All tests have passed with this driver: %v and this scheduler: %v\n",
		os.Args[2],
		os.Args[1],
	)
}