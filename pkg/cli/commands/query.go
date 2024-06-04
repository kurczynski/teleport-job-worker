package commands

import (
	"context"
	"flag"
	"fmt"
	"github.com/kurczynski/teleport-job-worker/api/proto/job"
	"github.com/kurczynski/teleport-job-worker/internal/logging"
	"os"
)

type QueryCmd struct {
	client job.JobClient

	jobID string
}

func (s *QueryCmd) SetClient(client job.JobClient) {
	s.client = client
}

func (s *QueryCmd) ParseCLI(set *flag.FlagSet) error {
	idArg := set.String("id", "", "ID of the job to query")

	if err := parseOSArgs(set); err != nil {
		return err
	}

	s.jobID = *idArg

	return nil
}

func (s *QueryCmd) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCtxTimeout)
	defer cancel()

	resp, err := s.client.Query(ctx, &job.QueryRequest{Id: s.jobID})

	if err != nil {
		fmt.Println(err)

		os.Exit(1)
	}

	fmt.Println(resp)

	logging.Log.Debug("Query response", "response", resp)
}
