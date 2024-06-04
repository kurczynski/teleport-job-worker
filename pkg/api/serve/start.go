package serve

import (
	"context"
	jobproto "github.com/kurczynski/teleport-job-worker/api/proto/job"
	"github.com/kurczynski/teleport-job-worker/internal/logging"
	"github.com/kurczynski/teleport-job-worker/pkg/joblib/cgroups"
	"github.com/kurczynski/teleport-job-worker/pkg/joblib/jobs"
)

func (s *JobServer) Start(ctx context.Context, req *jobproto.StartRequest) (*jobproto.Response, error) {
	logging.Log.Debug("Handling start job request", "request", req)

	// TODO: The request should be validated before creating a job
	job, err := jobs.NewJob(s.WorkerName, s.Clock, getResourceLimits(req), req.Command.Name, req.Command.Args...)

	if err != nil {
		return nil, err
	}

	err = job.Start()

	if err != nil {
		return nil, err
	}

	s.Jobs[job.ID()] = job

	pb := ProtoBuf{}

	return &jobproto.Response{
		Info:           pb.toJobInfo(job),
		ResourceLimits: pb.toResources(job.Limits()),
	}, nil
}

func getResourceLimits(req *jobproto.StartRequest) cgroups.Resources {
	return cgroups.Resources{
		CPUPercentage: req.ResourceLimits.CpuPercentage,
		DiskIOBPS:     req.ResourceLimits.DiskIoBps,
		MemoryBytes:   req.ResourceLimits.MemoryBytes,
	}
}
