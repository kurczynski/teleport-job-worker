package jobs

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/kurczynski/teleport-job-worker/internal/clock"
	"github.com/kurczynski/teleport-job-worker/pkg/joblib/cgroups"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
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

const (
	ReadyStatus     = Status("ready")
	RunningStatus   = Status("running")
	StoppedStatus   = Status("stopped")
	FailedStatus    = Status("failed")
	SucceededStatus = Status("succeeded")
)

// Status Status of the job.
type Status string

// Job Contains information to interact with jobs.
type Job struct {
	id             string
	created        time.Time
	status         Status
	statusChanges  []StatusChange
	command        *exec.Cmd
	resourceLimits cgroups.Resources
	cgroup         *cgroups.Cgroup
	clock          clock.Clock
	stdoutFilename string
	stderrFilename string
}

// StatusChange When the status of the job was changed.
type StatusChange struct {
	Status    Status
	ChangedAt time.Time
}

// NewJob Create a new job to run the specified command using the given resource limits.
func NewJob(workerName string, clock clock.Clock, resourceLimits cgroups.Resources, command string, args ...string) (*Job, error) {
	logger.Debug("Creating new job", "workerName", workerName, "resourceLimits", resourceLimits, "command", command, "args", args)

	id := uuid.NewString()
	cg, err := cgroups.NewCgroup("/sys/fs/cgroup", workerName, id)

	if err != nil {
		return nil, err
	}

	cmd := exec.Command(command, args...)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		CgroupFD:    cg.FD(),
		Pdeathsig:   syscall.SIGKILL,
		Setpgid:     true,
		UseCgroupFD: true,
	}

	job := Job{
		id:             id,
		command:        cmd,
		created:        clock.Now(),
		clock:          clock,
		statusChanges:  make([]StatusChange, 0),
		resourceLimits: resourceLimits,
		cgroup:         cg,
		stdoutFilename: strings.Join([]string{"/tmp", fmt.Sprintf("%s-%s", id, "stdout")}, string(os.PathSeparator)),
		stderrFilename: strings.Join([]string{"/tmp", fmt.Sprintf("%s-%s", id, "stderr")}, string(os.PathSeparator)),
	}

	job.updateStatus(ReadyStatus)

	return &job, nil
}

// ID Returns the ID of the job.
func (j *Job) ID() string {
	return j.id
}

// Limits Returns the resource limits of the job.
func (j *Job) Limits() cgroups.Resources {
	return j.resourceLimits
}

// Command Returns the job's command with its arguments.
func (j *Job) Command() (string, []string) {
	return j.command.Path, j.command.Args
}

// Created Returns when the job was initially created.
func (j *Job) Created() time.Time {
	return j.created
}

// Status Returns the current status of the job.
func (j *Job) Status() Status {
	return j.status
}

// StatusChanges Returns what status changes the job has gone through along with a timestamp of when.
func (j *Job) StatusChanges() []StatusChange {
	return j.statusChanges
}

// Start Begin execution of the job's command immediately.
func (j *Job) Start() error {
	logger.Info("Starting job", "id", j.id, "command", j.command)

	if err := j.cgroup.Configure(j.resourceLimits); err != nil {
		j.updateStatus(FailedStatus)

		return err
	}

	go func() {
		runtime.LockOSThread()

		defer runtime.UnlockOSThread()
		defer j.cgroup.Cleanup()

		stdout, err := os.OpenFile(j.stdoutFilename, os.O_WRONLY|os.O_CREATE, 0644)

		if err != nil {
			logger.Error("Failed to open stdout", "err", err)
			j.updateStatus(FailedStatus)

			return
		}

		defer stdout.Close()

		stderr, err := os.OpenFile(j.stderrFilename, os.O_WRONLY|os.O_CREATE, 0644)

		if err != nil {
			logger.Error("Failed to open stderr", "err", err)
			j.updateStatus(FailedStatus)

			return
		}

		defer stderr.Close()

		j.command.Stdout = stdout
		j.command.Stderr = stderr

		if err := j.command.Start(); err != nil {
			logger.Error("Failed to start job", "err", err)
			j.updateStatus(FailedStatus)

			return
		}

		j.updateStatus(RunningStatus)

		logger.Debug("Started job command", "pid", j.command.Process.Pid)

		if err := j.command.Wait(); err != nil {
			logger.Error("Failed waiting for command to finish", "err", err)
			j.updateStatus(FailedStatus)

			return
		}

		j.updateStatus(SucceededStatus)
	}()

	return nil
}

// Stop End execution of the job immediately.
func (j *Job) Stop() error {
	logger.Info("Stopping job", "id", j.id, "command", j.command)

	defer j.cgroup.Cleanup()
	defer j.cleanup()

	if err := j.command.Process.Kill(); err != nil {
		j.updateStatus(FailedStatus)

		return err
	} else {
		j.updateStatus(StoppedStatus)

		return nil
	}
}

// Output Get the full output (stdout and stderr) from the job.
func (j *Job) Output() (*os.File, *os.File, error) {
	logger.Debug("Getting job output")

	var stdout *os.File
	var stderr *os.File

	if fout, err := os.Open(j.stdoutFilename); err != nil {
		return nil, nil, err
	} else {
		stdout = fout
	}

	logger.Debug("Opened stdout", "path", stdout.Name())

	if ferr, err := os.Open(j.stderrFilename); err != nil {
		return nil, nil, err
	} else {
		stderr = ferr
	}

	logger.Debug("Opened stderr", "path", stderr.Name())

	return stdout, stderr, nil
}

// updateStatus Update the job's status and record the time when it changed.
func (j *Job) updateStatus(status Status) {
	now := j.clock.Now()
	j.status = status

	j.statusChanges = append(j.statusChanges, StatusChange{Status: status, ChangedAt: now})
}

// cleanup Cleanup files used to store output from the job's command.
func (j *Job) cleanup() {
	if err := os.Remove(j.stdoutFilename); err != nil {
		logger.Error("Failed to remove stdout", "path", j.stdoutFilename)
	}

	if err := os.Remove(j.stderrFilename); err != nil {
		logger.Error("Failed to remove stderr", "path", j.stderrFilename)
	}
}
