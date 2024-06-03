package commands

import (
	"flag"
	"github.com/kurczynski/teleport-job-worker/api/proto/job"
)

type QueryCmd struct {
	client job.JobClient
}

func (s *QueryCmd) SetClient(client job.JobClient) {
	s.client = client
}

func (s *QueryCmd) ParseCLI(set *flag.FlagSet) error {
	return nil
}

func (s *QueryCmd) Run() {

}
