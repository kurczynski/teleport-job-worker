package commands

import (
	"context"
	"flag"
	"fmt"
	"github.com/kurczynski/teleport-job-worker/api/proto/job"
	"github.com/kurczynski/teleport-job-worker/internal/logging"
	"os"
)

type StopCmd struct {
	client job.JobClient

	jobID string
}

func (s *StopCmd) SetClient(client job.JobClient) {
	s.client = client
}

func (s *StopCmd) ParseCLI(set *flag.FlagSet) error {
	idArg := set.String("id", "", "ID of the job to query")

	if err := parseOSArgs(set); err != nil {
		return err
	}

	s.jobID = *idArg

	return nil
}

func (s *StopCmd) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCtxTimeout)
	defer cancel()

	resp, err := s.client.Stop(ctx, &job.StopRequest{Id: s.jobID})

	if err != nil {
		fmt.Println(err)

		os.Exit(1)
	}

	fmt.Println(resp.String())

	logging.Log.Debug("Stop response", "response", resp)
}
