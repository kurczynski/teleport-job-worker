package joblib

import (
	"fmt"
	"github.com/google/uuid"
	"io"
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

const (
	CgroupRoot = "/sys/fs/cgroup"

	ExitSuccess      = 0
	ExitProcessError = 1
)

// Job Contains information to interact with jobs.
type Job struct {
	id             string
	processCmd     *exec.Cmd
	forkCmd        *exec.Cmd
	resourceLimits Resources
	cgroup         *cgroup
	stdout         io.Reader
	stderr         io.Reader
}

// Resources cgroup limits that can be configured for jobs.
type Resources struct {
	MemoryBytes   uint64
	CPUPercentage int
	DiskIOBps     int
}

// cgroup Information used to construct a job's cgroup.
type cgroup struct {
	root       string
	workerName string
	jobID      string
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
		logger.Error("Failed to create cgroup", "err", err)

		return nil, err
	}

	return cg, nil
}

// NewJob Create a new job to run the specified command using the given resource limits.
func NewJob(workerName string, resourceLimits Resources, command string, args ...string) (*Job, error) {
	logger.Debug("Creating new job", "workerName", workerName, "resourceLimits", resourceLimits, "command", command, "args", args)

	id := uuid.NewString()
	cg, err := newCgroup(CgroupRoot, workerName, id)

	if err != nil {
		return nil, err
	}

	return &Job{
		id:             id,
		processCmd:     exec.Command(command, args...),
		forkCmd:        exec.Command("/proc/self/exe", "job"),
		resourceLimits: resourceLimits,
		cgroup:         cg,
	}, nil
}

// ID Returns the ID of the job.
func (j *Job) ID() string {
	return j.id
}

// Start Begin execution of the job's command immediately.
func (j *Job) Start() {
	logger.Debug("Starting job", "args", os.Args, "ppid", os.Getppid(), "pid", os.Getpid())

	if len(os.Args) > 1 && os.Args[1] == "job" {
		logger.Info("Starting job process", "id", j.id, "process", j.processCmd.String())

		j.runProcess()
	} else {
		logger.Info("Starting job", "id", j.id)

		j.fork()
	}
}

// Stop End execution of the job immediately.
// FIXME: Handle the stop signal so that the job does not exit as a failure
func (j *Job) Stop() error {
	logger.Info("Stopping job", "id", j.id)

	return j.forkCmd.Process.Signal(syscall.SIGTERM)
}

// Output Get the full output (stdout and stderr) from the job.
func (j *Job) Output() (io.Reader, io.Reader) {
	return j.stdout, j.stderr
}

// fork Fork from the main execution process and store the child's output pipes to be referenced in the parent process.
func (j *Job) fork() {
	rStdout, wStdout, err := os.Pipe()

	if err != nil {
		logger.Error("Failed to create stdout pipe", "err", err)

		return
	}

	rStderr, wStderr, err := os.Pipe()

	if err != nil {
		logger.Error("Failed to create stderr pipe", "err", err)

		return
	}

	j.stdout = rStdout
	j.stderr = rStderr

	go func() {
		defer wStdout.Close()
		defer wStderr.Close()

		j.forkCmd.Stdout = wStdout
		j.forkCmd.Stderr = wStderr
		j.forkCmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		}

		if err := j.forkCmd.Start(); err != nil {
			logger.Error("Failed to fork job", "id", j.id, "err", err)

			return
		}

		logger.Debug("Waiting for job to run", "id", j.id)

		if err := j.forkCmd.Wait(); err != nil {
			logger.Error("Job failed", "id", j.id, "err", err)

			return
		}

		logger.Info("Job completed successfully", "id", j.id)
	}()
}

// runProcess run the process that the job owns.
func (j *Job) runProcess() {
	pid := os.Getpid()

	if err := j.cgroup.setup(pid, j.resourceLimits); err != nil {
		logger.Error("Cgroup setup failed", "err", err)

		os.Exit(ExitProcessError)
	}

	j.processCmd.Stdin = os.Stdin
	j.processCmd.Stdout = os.Stdout
	j.processCmd.Stderr = os.Stderr

	if err := j.processCmd.Start(); err != nil {
		logger.Error("Failed to start job process", "id", j.id, "process", j.processCmd.String())

		os.Exit(ExitProcessError)
	}

	if err := j.processCmd.Wait(); err != nil {
		logger.Error("Job process failed", "id", j.id, "err", err)

		os.Exit(ExitProcessError)
	}

	logger.Debug("Job process completed successfully")

	os.Exit(ExitSuccess)
}

// setup Restricts the given PID to the given resource limits.
func (c *cgroup) setup(pid int, resourceLimits Resources) error {
	if err := c.setSubtreeController("+memory +cpu +io"); err != nil {
		return err
	}

	if err := c.setMemory(resourceLimits.MemoryBytes); err != nil {
		return err
	}

	if err := c.setCPU(resourceLimits.CPUPercentage); err != nil {
		return err
	}

	if err := c.setDiskIO(); err != nil {
		return err
	}

	if err := c.addProc(pid); err != nil {
		return err
	}

	return nil
}

// addProc Adds the given PID to the list of processes to have their resources managed.
func (c *cgroup) addProc(pid int) error {
	f, err := os.OpenFile(c.withJobPath("cgroup.procs"), os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	defer f.Close()

	if err := c.setResource(f, strconv.Itoa(pid)); err != nil {
		return err
	}

	return nil
}

// setMemory Set the maximum amount of memory the job can use.
// TODO: Double check the units used to limit memory (defined in kB?)
func (c *cgroup) setMemory(memoryMax uint64) error {
	f, err := os.OpenFile(c.withJobPath("memory.max"), os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	defer f.Close()

	if err := c.setResource(f, strconv.FormatUint(memoryMax, 10)); err != nil {
		return err
	}

	return nil
}

// setCPU Set the maximum amount of CPU the job can use as a percentage.
func (c *cgroup) setCPU(percentage int) error {
	period := 1 * time.Second
	quota := period.Microseconds() * int64(percentage)

	f, err := os.OpenFile(c.withJobPath("cpu.max"), os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	defer f.Close()

	return c.setResource(f, fmt.Sprintf("%d %d\n", quota, period.Microseconds()))
}

// setDiskIO Set the maximum amount of disk IO the job can use.
// TODO: Implement
func (c *cgroup) setDiskIO() error {
	return nil
}

// setResource Generalized function to help set resources in a unified way.
func (c *cgroup) setResource(resource io.Writer, value string) error {
	n, err := resource.Write([]byte(value))

	if err != nil {
		return err
	}

	logger.Debug("Set resource", "value", value, "bytesWritten", n)

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
