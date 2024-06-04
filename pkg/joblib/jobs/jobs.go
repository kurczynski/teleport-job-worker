package jobs

import (
	"github.com/google/uuid"
	"github.com/kurczynski/teleport-job-worker/internal/clock"
	"github.com/kurczynski/teleport-job-worker/pkg/joblib/cgroups"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
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
	stdout         *io.PipeReader
	stderr         *io.PipeReader

	stdoutBufWrite *io.PipeWriter
	stdoutBufRead  *io.PipeReader
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

	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()

	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

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
		stdout:         stdoutReader,
		stderr:         stderrReader,
	}

	job.updateStatus(ReadyStatus)

	return &job, nil
}

// ID Returns the ID of the job.
func (j *Job) ID() string {
	return j.id
}

// Command Returns the job's command with its arguments.
func (j *Job) Command() (string, []string) {
	return j.command.Path, j.command.Args
}

func (j *Job) Created() time.Time {
	return j.created
}

func (j *Job) Status() Status {
	return j.status
}

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

		if err := j.command.Start(); err != nil {
			logger.Error("Failed to start job", "err", err)
			j.updateStatus(FailedStatus)

			return
		}

		j.updateStatus(RunningStatus)

		logger.Debug("Started job command", "pid", j.command.Process.Pid)

		// TODO: Check how this interacts with piped stdout and stderr
		err := j.command.Wait()

		if err != nil {
			logger.Error("Failed waiting for command to finish", "err", err)
			j.updateStatus(FailedStatus)

			return
		}

		err = j.stdout.Close()
		if err != nil {
			log.Fatalf("stdout pipe close failed: %s\n", err)
			return
		}

		err = j.stderr.Close()
		if err != nil {
			log.Fatalf("stderr pipe close failed: %s\n", err)
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

	if err := j.command.Process.Kill(); err != nil {
		j.updateStatus(FailedStatus)

		return err
	} else {
		j.updateStatus(StoppedStatus)

		return nil
	}
}

// Output Get the full output (stdout and stderr) from the job.
func (j *Job) Output() (io.Reader, io.Reader) {
	return j.stdout, j.stderr
}

// updateStatus Update the job's status and record the time when it changed.
func (j *Job) updateStatus(status Status) {
	now := j.clock.Now()
	j.status = status

	j.statusChanges = append(j.statusChanges, StatusChange{Status: status, ChangedAt: now})
}
