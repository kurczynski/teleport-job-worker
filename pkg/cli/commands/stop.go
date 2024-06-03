package commands

import (
	"flag"
	"github.com/kurczynski/teleport-job-worker/api/proto/job"
)

type StopCmd struct {
	client job.JobClient
}

func (s *StopCmd) SetClient(client job.JobClient) {
	s.client = client
}

func (s *StopCmd) ParseCLI(set *flag.FlagSet) error {
	return nil
}

func (s *StopCmd) Run() {

}
