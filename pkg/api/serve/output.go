package serve

import (
	"bufio"
	"errors"
	jobproto "github.com/kurczynski/teleport-job-worker/api/proto/job"
	"github.com/kurczynski/teleport-job-worker/internal/logging"
	"log"
)

func (s *JobServer) Output(req *jobproto.OutputRequest, stream jobproto.Job_OutputServer) error {
	logging.Log.Debug("Handling output request", "request", req)

	job, ok := s.Jobs[req.Id]

	if !ok {
		err := errors.New("job does not exist")
		logging.Log.Error("Failed to open job output", "err", err)

		return err
	}

	stdout, _, err := job.Output()

	if err != nil {
		logging.Log.Error("Failed to open job output", "err", err)

		return err
	}

	defer stdout.Close()

	log.Printf("Let's read some output from: %s\n", stdout.Name())

	reader := bufio.NewReader(stdout)
	buf := make([]byte, 0, 1024)
	var resp *jobproto.OutputResponse

	for {
		if n, err := reader.Read(buf); err != nil {
			return err
		} else {
			logging.Log.Debug("Wrote to buffer", "bytes", n)

			if n <= 0 {
				break
			}
		}

		resp = &jobproto.OutputResponse{
			Stdout: buf,
			Stderr: nil,
		}

		if err := stream.Send(resp); err != nil {
			return err
		}
	}

	return nil
}
