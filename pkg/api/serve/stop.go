package serve

import (
	"context"
	jobproto "github.com/kurczynski/teleport-job-worker/api/proto/job"
	"github.com/kurczynski/teleport-job-worker/internal/logging"
)

func (s *JobServer) Stop(ctx context.Context, req *jobproto.StopRequest) (*jobproto.Response, error) {
	logging.Log.Debug("Handling stop job request", "request", req)

	job, ok := s.Jobs[req.Id]

	if !ok {
		return nil, nil
	}

	err := job.Stop()

	if err != nil {
		return nil, err
	}

	pb := ProtoBuf{}

	return &jobproto.Response{
		Info:           pb.toJobInfo(job),
		ResourceLimits: pb.toResources(job.Limits()),
	}, nil
}
