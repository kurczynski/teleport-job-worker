package commands

import (
	"flag"
	"github.com/kurczynski/teleport-job-worker/api/proto/job"
)

type OutputCmd struct {
	client job.JobClient
}

func (s *OutputCmd) SetClient(client job.JobClient) {
	s.client = client
}

func (s *OutputCmd) ParseCLI(set *flag.FlagSet) error {
	return nil
}

func (s *OutputCmd) Run() {

}
