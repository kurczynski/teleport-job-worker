package commands

import (
	"context"
	"flag"
	"fmt"
	"github.com/kurczynski/teleport-job-worker/api/proto/job"
	"github.com/kurczynski/teleport-job-worker/internal/logging"
	"os"
)

type OutputCmd struct {
	client job.JobClient

	jobID string
}

func (s *OutputCmd) SetClient(client job.JobClient) {
	s.client = client
}

func (s *OutputCmd) ParseCLI(set *flag.FlagSet) error {
	idArg := set.String("id", "", "ID of the job to query")

	if err := parseOSArgs(set); err != nil {
		return err
	}

	s.jobID = *idArg

	return nil
}

func (s *OutputCmd) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCtxTimeout)
	defer cancel()

	stream, err := s.client.Output(ctx, &job.OutputRequest{Id: s.jobID})

	if err != nil {
		fmt.Println(err)

		os.Exit(1)
	}

	for {
		out, err := stream.Recv()

		if err != nil {
			logging.Log.Error("Failed to read stream", "err", err)

			return
		}

		fmt.Printf("%s", string(out.Stdout))
	}
}
