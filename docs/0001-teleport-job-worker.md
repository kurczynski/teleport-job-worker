---
authors: Brodie Kurczynski (brodie@kurczynski.org)
state: draft
---

# RFD 1 - Teleport Job Worker

## What

A service that runs commands submitted by multiple users. These commands may run concurrently and will provide the users
with the ability to limit resource (i.e. CPU, memory, and disk IO) usage. The service will consist of three pieces:

* A reusable library implementing the functionality for working with jobs
* An API server that wraps the functionality of the library
* A command line interface (CLI) that communicates with the API server

### Required Approvers

* Engineering: (@sclevine && @gavinfrazar) || (@sclevine && @bernardjkim) || (@sclevine && @jimbishopp) || (
  @gavinfrazar && @bernardjkim) || (@gavinfrazar @jimbishopp) || (@bernardjkim @jimbishopp)

### Security

Requests made to the service will use mTLS for authentication. Specifically:

* TLS using the modern version 1.3
* The cipher suite will need no configuration, it will be handled by `crypto/tls` (this was updated
  in [Go 1.17](https://go.dev/doc/go1.17#crypto/tls))
* X.509 public/private key pairs will be generated using RSA-4096; these can be generated using
  the [Makefile](../Makefile) to run `make create-certs`

For authorization, the service will ensure that users are only allowed to interact with the jobs they have created.

### UX

The CLI used in the service will be a simple application that implements the shared library and expose all the
functionality of the [API](#api). This is what they will look like in the CLI:

* **Start**
  ```shell
  job-worker start -cpu 10 -memory 1000 -io 400 -cmd "/usr/bin/sleep" -args "60"
  ```
  The auto-generated ID for the job will be returned if the job is started successfully. A UUID will be used as the
  job's ID because it will allow for an effectively unlimited number of jobs to be identified uniquely without having to
  manage or store any additional states. The resource arguments use the following units:
    * `cpu` percentage of available CPU
    * `memory` bytes
    * `io` bytes per second limit on reads and bytes per second limit on writes (e.g. 400 will limit reads to 400 bytes
      per second and writes to 400 bytes per second)
* **Stop**
  ```shell
  job-worker stop -id "some-job-id"
  ```
* **Query**
  ```shell
  job-worker query -id "some-job-id"
  ```
  Returns details about the job that matches the provided ID.
* **Output**
  ```shell
  job-worker output -id "some-job-id"
  ```
  Returns all the output for the job that matches the provided ID.

Note that paths used for certificate files in the CLI can be configured setting these environment variables:

* `CLI_CERT_DIR`
* `CLI_CERT_FILE`
* `CLI_KEY_FILE`

Their default values can be found [here](../pkg/config/certs.go).

### Process Execution

Jobs will be forked from the service's process and contain exactly one process that it manages. A job must have one of
the statuses defined in `Statuses` of the [proto spec](../api/proto/job/job.proto) and will transition between statuses
as show below:

```
  ┌───────────────────────────┐
  │                           │
ready      ┌────►stopped◄─────┤
  │        │                  │
  │        │                  │
  │        ├────►failed◄──────┘
  │        │                   
  ▼        │                   
running────┴────►succeeded     
```

Resource control will be implemented using cgroups. A parent directory `job-worker` will be created
under `/sys/fs/cgroup` that will contain a directory for every job created. For example, the path for `job-a` would look
like `/sys/fs/cgroup/job-worker/job-a`. Under this path, the following files would be updated to manage the resources
for the job:

* `cgroup.procs` contains the PID of the job, i.e. the forked process; cgroups will be applied to this PID in order to
  control the resources of all child processes
* `cpu.max` limits CPU usage
* `memory.max` limits memory usage
* `io.max` limits disk IO

Note that for this implementation, job metadata will remain in memory until the application exits. A real implementation
of this service would store job metadata in some kind of persistent storage like a database.

### Proto Specification

The service will implement the following protobufs:

* [job](../api/proto/job/job.proto)

### API

The service's API will contain the resources listed below and be made available using gRPC. The API has settings that
can be managed in its [configuration file](../config/server.json).

#### job

A job is responsible for managing the lifecycle of the process given to it.

* `Start` start a new job and begin execution of the specified command immediately
* `Stop` stop execution of the specified job immediately
* `Query` query details about specified job; this function can run on a job of any status
* `Output` get the full output (stdout and stderr) of any existing job; the output from the job will be written to
  memory as the process executes so that it can be retrieved by calls to this function; for this exercise, output will
  remain available until the job worker has exited; for real world uses, the output would need to be removed or written
  to persistent storage because of memory limitations

Details of a job's functions are listed in [job.proto](../api/proto/job/job.proto).

### Library Godoc

```
package library // import "github.com/kurczynski/teleport-job-worker/pkg/joblib/jobs"

TYPES

// Status Status of the job.
type Status string

// Job Contains information to interact with jobs.
type Job struct {
	// Has unexported fields.
}

// StatusChange When the status of the job was changed.
type StatusChange struct {
	Status    Status
	ChangedAt time.Time
}

// Resources cgroup limits that can be configured for jobs.
type Resources struct {
	MemoryBytes   uint64
	CPUPercentage int32
	DiskIOBps     int32
}

// NewJob Create a new job to run the specified command using the given resource limits.
func NewJob(workerName string, clock clock.Clock, resourceLimits cgroups.Resources, command string, args ...string) (*Job, error)

// ID Returns the ID of the job.
func (j *Job) ID() string

// Limits Returns the resource limits of the job.
func (j *Job) Limits() cgroups.Resources

// Command Returns the job's command with its arguments.
func (j *Job) Command() (string, []string)

// Created Returns when the job was initially created.
func (j *Job) Created() time.Time

// Status Returns the current status of the job.
func (j *Job) Status() Status

// StatusChanges Returns what status changes the job has gone through along with a timestamp of when.
func (j *Job) StatusChanges() []StatusChange

// Output Get the full output (stdout and stderr) from the job.
func (j *Job) Output() *job.OutputRequest

// Start Begin execution of the job's command immediately.
func (j *Job) Start() error

// Stop End execution of the job immediately.
func (j *Job) Stop()
```
```
package library // import "github.com/kurczynski/teleport-job-worker/pkg/joblib/cgroups"

TYPES

// Cgroup Information used to construct cgroup.
type Cgroup struct {
	// Has unexported fields.
}

// Resources cgroup limits that can be configured for jobs.
type Resources struct {
	CPUPercentage int32
	DiskIOBPS     int32
	MemoryBytes   uint64
}

// NewCgroup Creates a new cgroup for the given job.
func NewCgroup(cgroupRoot string, workerName string, jobID string) (*Cgroup, error)

// FD Returns the cgroup file descriptor.
func (c *Cgroup) FD() int

// Configure Configure a cgroup with the given resource limits.
func (c *Cgroup) Configure(resourceLimits Resources) error

// Cleanup Remove cgroup files created for the job.
func (c *Cgroup) Cleanup()
```

### Observability

For the purpose of this challenge, simple logging will be the only observability implemented. If this service were to be
used in production, these are some of the observability features that would be useful.

#### Metrics

* `job_status`
* `start_job`
* `stop_job`
* `query_job`
* `job_output`

#### Events

* `JobStartTime`
* `JobFinishTime`

### Test Plan

Unit testing for this service will be fairly limited since the functionality of resource limits is handled by the
kernel. However, unit tests should be created to test the information that the service provides to the kernel. For
example, the input to a function that limits memory usage should be validated with its output that is passed to the
kernel.

Integration testing would be useful to validate the service's functionality against changes in the kernel. Although
breaking changes in cgroups _shouldn't_ happen, integration tests would help further ensure this. For example, the input
to a function that limits memory usage should be validated with how much memory is actually available to the job.

Finally, end-to-end testing would be useful to help ensure the user experience remains as expected. For example, the
input for creating a job should be validated with the output of a job.
