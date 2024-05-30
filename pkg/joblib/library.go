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
	ExitSuccess      = 0
	ExitProcessError = 1
	ExitIOError      = 2
)

// XXX: /proc/self/limits might be of use at some point

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

type cgroup struct {
	workerName string
	paths      cgroupPaths
}

type cgroupPaths struct {
	root   string
	worker string
	job    string
}

func newCgroup(cgroupRoot string, workerName string, jobID string) (*cgroup, error) {
	workerPath := strings.Join([]string{cgroupRoot, workerName}, string(os.PathSeparator))
	jobPath := strings.Join([]string{workerPath, jobID}, string(os.PathSeparator))

	logger.Debug("Creating new cgroup", "path", jobPath)

	err := os.MkdirAll(jobPath, 0644)

	if err != nil {
		logger.Error("Failed to create cgroup", "err", err)

		return nil, err
	}

	return &cgroup{
		workerName: workerName,
		paths: cgroupPaths{
			root:   cgroupRoot,
			worker: workerPath,
			job:    jobPath,
		},
	}, nil
}

// NewJob Create a new job to run the specified command using the given resource limits.
func NewJob(worker string, resourceLimits Resources, command string, args ...string) (*Job, error) {
	logger.Debug("Creating new job", "worker", worker, "resourceLimits", resourceLimits, "command", command, "args", args)

	id := uuid.NewString()
	cg, err := newCgroup("/sys/fs/cgroup", worker, id)

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

func (j *Job) fork() {
	rStdout, wStdout, err := os.Pipe()

	if err != nil {
		logger.Error("Failed to create stdout pipe", "err", err)

		os.Exit(ExitIOError)
	}

	rStderr, wStderr, err := os.Pipe()

	if err != nil {
		logger.Error("Failed to create stderr pipe", "err", err)

		os.Exit(ExitIOError)
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

		err = j.forkCmd.Start()

		if err != nil {
			logger.Error("Failed to fork job", "id", j.id, "err", err)

			return
		}

		logger.Debug("Waiting for job to run", "id", j.id)

		err = j.forkCmd.Wait()

		if err != nil {
			logger.Error("Job failed", "id", j.id, "err", err)

			return
		}

		logger.Info("Job completed successfully", "id", j.id)
	}()
}

// runProcess run the process that the job owns.
func (j *Job) runProcess() {
	pid := os.Getpid()

	err := j.cgroup.setup(pid, j.resourceLimits)

	if err != nil {
		logger.Error("Setup failed", "err", err)

		os.Exit(ExitProcessError)
	}

	j.processCmd.Stdin = os.Stdin
	j.processCmd.Stdout = os.Stdout
	j.processCmd.Stderr = os.Stderr

	err = j.processCmd.Start()

	if err != nil {
		logger.Error("Failed to start job process", "id", j.id, "process", j.processCmd.String())

		os.Exit(ExitProcessError)
	}

	err = j.processCmd.Wait()

	if err != nil {
		logger.Error("Job process failed", "id", j.id, "err", err)

		os.Exit(ExitProcessError)
	}

	logger.Debug("Job process completed successfully")

	os.Exit(ExitSuccess)
}

// setup Restricts the given PID to the given resource limits.
func (c *cgroup) setup(pid int, resourceLimits Resources) error {
	// XXX: memory.max is defined in kB
	err := c.setMemory(resourceLimits.MemoryBytes)

	if err != nil {
		return err
	}

	err = c.setCpu(resourceLimits.CPUPercentage)

	if err != nil {
		return err
	}

	err = c.setIo()

	if err != nil {
		return err
	}

	err = c.addProc(pid)

	if err != nil {
		return err
	}

	return nil
}

func (c *cgroup) addProc(pid int) error {
	procPath := strings.Join([]string{c.paths.job, "cgroup.procs"}, string(os.PathSeparator))

	err := writeFile(procPath, strconv.Itoa(pid))

	if err != nil {
		return err
	}

	logger.Debug("Added process to cgroup", "pid", pid, "path", procPath)

	return nil
}

func (c *cgroup) setMemory(memoryMax uint64) error {
	err := c.setSubtreeController("+memory")

	if err != nil {
		return err
	}

	err = c.setResource("memory.max", strconv.FormatUint(memoryMax, 10))

	if err != nil {
		return err
	}

	return nil
}

func (c *cgroup) setCpu(percentage int) error {
	period := 1 * time.Second
	quota := period.Microseconds() * int64(percentage)

	err := c.setSubtreeController("+cpu")

	if err != nil {
		return err
	}

	err = c.setResource("cpu.max", fmt.Sprintf("%d %d\n", quota, period.Microseconds()))

	if err != nil {
		return err
	}

	return nil
}

// TODO: Implement
func (c *cgroup) setIo() error {
	err := c.setSubtreeController("+io")
	if err != nil {
		return err
	}

	return nil
}

func (c *cgroup) setResource(resource string, value string) error {
	resourcePath := strings.Join([]string{c.paths.job, resource}, string(os.PathSeparator))

	err := writeFile(resourcePath, value)

	if err != nil {
		return err
	}

	return nil
}

func (c *cgroup) setSubtreeController(arg string) error {
	/*
		Subtree controllers must be set one level up from the job due to the "no internal process constraint"
		https://www.kernel.org/doc/html/latest/admin-guide/cgroup-v2.html#no-internal-process-constraint
	*/

	// TODO: Check if the resource can be controlled
	// This lists the available resources that can be controlled
	_ = "/sys/fs/cgroup/cgroup.controllers"

	// Resources listed in this file allow containers to use them
	svcSubtree := strings.Join([]string{c.paths.worker, "cgroup.subtree_control"}, string(os.PathSeparator))

	// Use '+' to add resource and '-' to remove it
	return writeFile(svcSubtree, arg)
}

func writeFile(fpath string, arg string) error {
	f, err := os.OpenFile(fpath, os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	defer f.Close()

	if _, err = f.WriteString(fmt.Sprintf("%s", arg)); err != nil {
		return err
	}

	logger.Debug("Wrote to file", "file", f.Name(), "data", arg)

	return nil
}
