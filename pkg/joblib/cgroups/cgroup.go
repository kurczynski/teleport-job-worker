package cgroups

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	// TODO: This should be loaded or injected, not hardcoded
	logger = slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			AddSource: false,
			Level:     slog.LevelDebug,
		},
	))
)

// Cgroup Information used to construct cgroup.
type Cgroup struct {
	fd         int
	jobID      string
	root       string
	workerName string
}

// Resources cgroup limits that can be configured for jobs.
type Resources struct {
	CPUPercentage int32
	DiskIOBps     int32
	MemoryBytes   uint64
}

// partition Disk partition information.
type partition struct {
	major     string
	minor     string
	numBlocks string
	name      string
}

// NewCgroup Creates a new cgroup for the given job.
func NewCgroup(cgroupRoot string, workerName string, jobID string) (*Cgroup, error) {
	cg := &Cgroup{
		root:       cgroupRoot,
		workerName: workerName,
		jobID:      jobID,
	}

	jobPath := cg.withJobPath()

	logger.Debug("Creating new cgroup", "path", jobPath)

	if err := os.MkdirAll(jobPath, 0644); err != nil {
		logger.Error("Failed to create cgroup", "path", jobPath)

		return nil, err
	}

	if f, err := os.Open(jobPath); err != nil {
		return nil, err
	} else {
		cg.fd = int(f.Fd())
	}

	return cg, nil
}

// FD Returns the cgroup file descriptor.
func (c *Cgroup) FD() int {
	return c.fd
}

// Configure Configure a cgroup with the given resource limits.
func (c *Cgroup) Configure(resourceLimits Resources) error {
	if err := c.setSubtreeController("+memory +cpu +io"); err != nil {
		return err
	}

	if err := c.setMemory(resourceLimits.MemoryBytes); err != nil {
		return err
	}

	if err := c.setCPU(resourceLimits.CPUPercentage); err != nil {
		return err
	}

	partitions, err := getPartitions()

	if err != nil {
		return err
	}

	// For this exercise, let's just set the IO limit for reads and writes to all partitions.
	for _, part := range partitions {
		if err := c.setDiskIO(resourceLimits.DiskIOBps, part); err != nil {
			logger.Warn("Failed to set disk IO", "partition", part, "err", err)
		}
	}

	return nil
}

// Cleanup Remove cgroup files created for the job.
func (c *Cgroup) Cleanup() {
	if err := os.RemoveAll(c.withJobPath()); err != nil {
		logger.Error("Failed to cleanup cgroup", "err", err)

		return
	}

	logger.Debug("Cleaned up cgroup", "path", c.withJobPath())
}

// setMemory Set the maximum amount of memory the job can use in bytes.
func (c *Cgroup) setMemory(memoryMax uint64) error {
	if f, err := os.OpenFile(c.withJobPath("memory.max"), os.O_WRONLY, 0644); err != nil {
		return err
	} else {
		defer f.Close()

		value := strconv.FormatUint(memoryMax, 10)
		logger.Debug("Setting memory limit", "path", f.Name(), "value", value)

		return c.setResource(f, value)
	}
}

// setCPU Set the maximum amount of CPU the job can use as a percentage.
func (c *Cgroup) setCPU(percentage int32) error {
	period := 1 * time.Second
	quota := period.Microseconds() * int64(percentage)

	if f, err := os.OpenFile(c.withJobPath("cpu.max"), os.O_WRONLY, 0644); err != nil {
		return err
	} else {
		defer f.Close()

		value := fmt.Sprintf("%d %d\n", quota, period.Microseconds())
		logger.Debug("Setting CPU limit", "path", f.Name(), "value", value)

		return c.setResource(f, value)
	}
}

// setDiskIO Set the maximum amount of disk IO the job can use on a given partition.
func (c *Cgroup) setDiskIO(bytesSec int32, part partition) error {
	if part.minor != "0" {
		logger.Warn("Refusing to set IO limit, IO limits can only be set for physical devices", "partition", part)

		return nil
	}

	if f, err := os.OpenFile(c.withJobPath("io.max"), os.O_WRONLY, 0644); err != nil {
		return err
	} else {
		defer f.Close()

		value := fmt.Sprintf("%s:%s rbps=%d wbps=%d", part.major, part.minor, bytesSec, bytesSec)
		logger.Debug("Setting IO limit", "path", f.Name(), "value", value)

		return c.setResource(f, value)
	}
}

// getPartitions Get disk partitions available on the system.
func getPartitions() ([]partition, error) {
	f, err := os.Open("/proc/partitions")

	if err != nil {
		return nil, err
	}

	var partitions []partition

	scanner := bufio.NewScanner(f)

	// Skip the first two lines of the file, they're headers
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		partLine := strings.Fields(scanner.Text())

		part := partition{
			major:     partLine[0],
			minor:     partLine[1],
			numBlocks: partLine[2],
			name:      partLine[3],
		}

		logger.Debug("Found disk partition", "partition", partLine)

		partitions = append(partitions, part)
	}

	return partitions, nil
}

// setResource Generalized function to help set resources in a unified way.
func (c *Cgroup) setResource(resource io.Writer, value string) error {
	if _, err := resource.Write([]byte(value)); err != nil {
		return err
	}

	return nil
}

// setSubtreeController Sets the resources that can be limited in a job. The subtree control resources must be set
// before the job's resources limits can be set.
func (c *Cgroup) setSubtreeController(args ...string) error {
	// Subtree controllers must be set one level up from the job due to the "no internal process constraint"
	// https://www.kernel.org/doc/html/latest/admin-guide/cgroup-v2.html#no-internal-process-constraint
	f, err := os.OpenFile(c.withWorkerPath("cgroup.subtree_control"), os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	defer f.Close()

	for _, arg := range args {
		if err := c.setResource(f, arg); err != nil {
			return err
		}
	}

	return nil
}

// withWorkerPath Utility function to generating the worker's cgroup path easier.
func (c *Cgroup) withWorkerPath(resource ...string) string {
	return strings.Join(append([]string{c.root, c.workerName}, resource...), string(os.PathSeparator))
}

// withJobPath Utility function to generating the job's cgroup path easier.
func (c *Cgroup) withJobPath(resource ...string) string {
	return strings.Join(append([]string{c.root, c.workerName, c.jobID}, resource...), string(os.PathSeparator))
}
