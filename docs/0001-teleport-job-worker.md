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
* X.509 public/private key pairs will be generated using RSA-4096

For authorization, the service will use simple tokens to ensure that jobs can only be accessed and modified by the user
who created them. This will be done by adding `grpc.UnaryInterceptor` to the API server to check if the user has
authorization to interact with the resource. For the purposes of this challenge, this type of authorization will be
fine. However, proper OAuth would need to be implemented for use in production.

### UX

The CLI used in the service will be a simple application that implements the shared library and expose all the
functionality of the [API](#api). This is what they will look like in the CLI:

* **Start**
  ```shell
  job-worker -token "some-token" start -cpu 10 -memory 1000 -io 400 -cmd "/usr/bin/sleep" -args "60"
  ```
  The auto-generated ID for the job will be returned if the job is started successfully. The resource arguments use the
  following units:
    * `cpu` percentage of available CPU
    * `memory` bytes
    * `io` bytes per second limit on reads and bytes per second limit on writes (e.g. 400 will limit reads to 400 bytes
      per second and writes to 400 bytes per second)
* **Stop**
  ```shell
  job-worker -token "some-token" stop -id "some-job-id"
  ```
* **Query**
  ```shell
  job-worker -token "some-token" query -id "some-job-id"
  ```
  Returns details about the job that matches the provided ID.
* **Output**
  ```shell
  job-worker -token "some-token" output -id "some-job-id"
  ```
  Returns all the output for the job that matches the provided ID.

### Process Execution

Jobs will be forked from the service's process and contain exactly one process that it manages. A job must have one of
the statuses defined in `Statuses` of the [proto spec](../api/proto/job/job.proto) where it will start in an initial
state and finish in one of three terminal states:

```
            ┌──► stopped  
            │             
running ────┼──► failed   
            │             
            └──► succeeded
```

Resource control will be implemented using cgroups. A parent directory `job-worker` will be created
under `/sys/fs/cgroup` that will contain a directory for every job created. For example, the path for `job-a` would look
like `/sys/fs/cgroup/job-worker/job-a`. Under this path, the following files would be updated to manage the resources
for the job:

* `cgroup.procs` contains the PID of the job's process being managed
* `cpu.max` limits CPU usage
* `memory.max` limits memory usage
* `io.max` limits disk IO

Note that for this implementation, job metadata will remain in memory until the application exits. A real implementation
of this service would store job metadata in some kind of persistent storage like a database.

### Proto Specification

The service will implement the following protobufs:

* [job](../api/proto/job/job.proto)

### API

The service's API will contain the resources listed below and be made available using gRPC.

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
