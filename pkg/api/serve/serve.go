package serve

import (
	jobproto "github.com/kurczynski/teleport-job-worker/api/proto/job"
	"github.com/kurczynski/teleport-job-worker/internal/clock"
	"github.com/kurczynski/teleport-job-worker/pkg/joblib/cgroups"
	"github.com/kurczynski/teleport-job-worker/pkg/joblib/jobs"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type JobServer struct {
	WorkerName string
	Clock      clock.Clock

	Jobs map[string]*jobs.Job

	jobproto.UnimplementedJobServer
	credentials.TransportCredentials
}

// ProtoBuf Contains functions that convert types to protobufs.
type ProtoBuf struct{}

func (p *ProtoBuf) toJobInfo(job *jobs.Job) *jobproto.Info {
	command, args := job.Command()

	return &jobproto.Info{
		ID:           job.ID(),
		Status:       p.toStatus(job.Status()),
		Created:      timestamppb.New(job.Created()),
		StatusChange: p.toStatusChanges(job.StatusChanges()),
		Command:      &jobproto.Command{Name: command, Args: args},
	}
}

func (p *ProtoBuf) toStatus(status jobs.Status) jobproto.Status {
	switch string(status) {
	case jobproto.Status_READY.String():
		return jobproto.Status_READY
	}

	return jobproto.Status_RUNNING
}

func (p *ProtoBuf) toStatusChanges(statusChanges []jobs.StatusChange) []*jobproto.StatusChange {
	pbStatusChanges := make([]*jobproto.StatusChange, 0)

	for _, statusChange := range statusChanges {
		pbStatus := p.toStatus(statusChange.Status)

		pbStatusChange := &jobproto.StatusChange{
			Status:    pbStatus,
			ChangedAt: timestamppb.New(statusChange.ChangedAt),
		}

		pbStatusChanges = append(pbStatusChanges, pbStatusChange)
	}

	return pbStatusChanges
}

func (p *ProtoBuf) toResources(resources cgroups.Resources) *jobproto.Resources {
	return &jobproto.Resources{
		MemoryBytes:   resources.MemoryBytes,
		CpuPercentage: resources.CPUPercentage,
		DiskIoBps:     resources.DiskIOBPS,
	}
}
