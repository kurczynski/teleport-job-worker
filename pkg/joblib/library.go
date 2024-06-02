package joblib

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	// TODO: This should be loaded from some configuration and not hardcoded
	logger = slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			AddSource: false,
			Level:     slog.LevelDebug,
		},
	))
)

// Job Contains information to interact with jobs.
type Job struct {
	id             string
	command        *exec.Cmd
	resourceLimits Resources
	cgroup         *cgroup
	stdout         io.Reader
	stderr         io.Reader
	stdoutBuf      *bytes.Buffer // Used to persist stdout
	stderrBuf      *bytes.Buffer // Used to persist stderr
}

// Resources cgroup limits that can be configured for jobs.
type Resources struct {
	CPUPercentage int
	DiskIOBps     int
	MemoryBytes   uint64
}

// cgroup Information used to construct a job's cgroup.
type cgroup struct {
	fd         int
	jobID      string
	root       string
	workerName string
}

// partition Disk partition information.
type partition struct {
	major     string
	minor     string
	numBlocks string
	name      string
}

// newCgroup Creates a new cgroup for the given job.
func newCgroup(cgroupRoot string, workerName string, jobID string) (*cgroup, error) {
	cg := &cgroup{
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

// NewJob Create a new job to run the specified command using the given resource limits.
func NewJob(workerName string, resourceLimits Resources, command string, args ...string) (*Job, error) {
	logger.Debug("Creating new job", "workerName", workerName, "resourceLimits", resourceLimits, "command", command, "args", args)

	id := uuid.NewString()
	cg, err := newCgroup("/sys/fs/cgroup", workerName, id)

	if err != nil {
		return nil, err
	}

	cmd := exec.Command(command, args...)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:  syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		UseCgroupFD: true,
		CgroupFD:    cg.fd,
	}

	stdoutPipe, err := cmd.StdoutPipe()

	if err != nil {
		return nil, err
	}

	stderrPipe, err := cmd.StderrPipe()

	if err != nil {
		return nil, err
	}

	return &Job{
		id:             id,
		command:        cmd,
		resourceLimits: resourceLimits,
		cgroup:         cg,
		stdout:         stdoutPipe,
		stderr:         stderrPipe,
		stdoutBuf:      new(bytes.Buffer),
		stderrBuf:      new(bytes.Buffer),
	}, nil
}

// ID Returns the ID of the job.
func (j *Job) ID() string {
	return j.id
}

// Start Begin execution of the job's command immediately.
func (j *Job) Start() error {
	logger.Info("Starting job", "id", j.id, "command", j.command)

	if err := j.cgroup.configure(j.resourceLimits); err != nil {
		return err
	}

	if err := j.command.Start(); err != nil {
		return err
	}

	go func() {
		defer j.cgroup.cleanup()

		nStdout, err := j.stdoutBuf.ReadFrom(j.stdout)

		if err != nil {
			log.Fatalln(err)
		}

		logger.Debug("Read from stdout to buffer", "bytes", nStdout)

		nStderr, err := j.stderrBuf.ReadFrom(j.stderr)

		if err != nil {
			log.Fatalln(err)
		}

		logger.Debug("Read from stderr to buffer", "bytes", nStderr)
	}()

	return nil
}

// Stop End execution of the job immediately.
func (j *Job) Stop() error {
	logger.Info("Stopping job", "id", j.id, "command", j.command)

	defer j.cgroup.cleanup()

	return j.command.Process.Kill()
}

// Output Get the full output (stdout and stderr) from the job.
func (j *Job) Output() (io.Reader, io.Reader) {
	// TODO: This reader will only see what's in the buffer when it is called. If the buffer is updated, a new reader
	// will need to be created to view the content. This should be improved.
	return bytes.NewReader(j.stdoutBuf.Bytes()), bytes.NewReader(j.stderrBuf.Bytes())
}

// configure Configure a cgroup with the given resource limits.
func (c *cgroup) configure(resourceLimits Resources) error {
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

// cleanup Remove cgroup files created for the job.
func (c *cgroup) cleanup() {
	if err := os.RemoveAll(c.withJobPath()); err != nil {
		logger.Error("Failed to cleanup cgroup", "err", err)

		return
	}

	logger.Debug("Cleaned up cgroup", "path", c.withJobPath())
}

// setMemory Set the maximum amount of memory the job can use.
// TODO: Double check the units used to limit memory (defined in kB?)
func (c *cgroup) setMemory(memoryMax uint64) error {
	if f, err := os.OpenFile(c.withJobPath("memory.max"), os.O_WRONLY, 0644); err != nil {
		return err
	} else {
		defer f.Close()

		return c.setResource(f, strconv.FormatUint(memoryMax, 10))
	}
}

// setCPU Set the maximum amount of CPU the job can use as a percentage.
func (c *cgroup) setCPU(percentage int) error {
	period := 1 * time.Second
	quota := period.Microseconds() * int64(percentage)

	if f, err := os.OpenFile(c.withJobPath("cpu.max"), os.O_WRONLY, 0644); err != nil {
		return err
	} else {
		defer f.Close()

		return c.setResource(f, fmt.Sprintf("%d %d\n", quota, period.Microseconds()))
	}
}

// setDiskIO Set the maximum amount of disk IO the job can use on a given partition.
func (c *cgroup) setDiskIO(bytesSec int, part partition) error {
	if part.minor != "0" {
		logger.Warn("Refusing to set IO limit, IO limits can only be set for physical devices", "partition", part)

		return nil
	}

	value := fmt.Sprintf("%s:%s rbps=%d wbps=%d", part.major, part.minor, bytesSec, bytesSec)

	if f, err := os.OpenFile(c.withJobPath("io.max"), os.O_WRONLY, 0644); err != nil {
		return err
	} else {
		defer f.Close()

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
func (c *cgroup) setResource(resource io.Writer, value string) error {
	if _, err := resource.Write([]byte(value)); err != nil {
		return err
	}

	return nil
}

// setSubtreeController Sets the resources that can be limited in a job. The subtree control resources must be set
// before the job's resources limits can be set.
func (c *cgroup) setSubtreeController(args ...string) error {
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
func (c *cgroup) withWorkerPath(resource ...string) string {
	return strings.Join(append([]string{c.root, c.workerName}, resource...), string(os.PathSeparator))
}

// withJobPath Utility function to generating the job's cgroup path easier.
func (c *cgroup) withJobPath(resource ...string) string {
	return strings.Join(append([]string{c.root, c.workerName, c.jobID}, resource...), string(os.PathSeparator))
}
