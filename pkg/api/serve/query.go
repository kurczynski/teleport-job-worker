package serve

import (
	"context"
	jobproto "github.com/kurczynski/teleport-job-worker/api/proto/job"
	"github.com/kurczynski/teleport-job-worker/internal/logging"
)

func (s *JobServer) Query(ctx context.Context, req *jobproto.QueryRequest) (*jobproto.Response, error) {
	logging.Log.Debug("Handling query request", "request", req)

	job, ok := s.Jobs[req.Id]

	if !ok {
		return nil, nil
	}

	pb := ProtoBuf{}

	return &jobproto.Response{
		Info:           pb.toJobInfo(job),
		ResourceLimits: nil,
	}, nil
}
