package commands

import (
	"context"
	"flag"
	"fmt"
	"github.com/kurczynski/teleport-job-worker/api/proto/job"
	"os"
	"strings"
)

type StartCmd struct {
	client job.JobClient

	jobCommand  string
	args        []string
	memoryLimit uint64
	cpuLimit    int32
	diskIOLimit int32
}

func (s *StartCmd) SetClient(client job.JobClient) {
	s.client = client
}

func (s *StartCmd) ParseCLI(set *flag.FlagSet) error {
	jobCommandArg := set.String("command", "", "job command to run")
	argsArg := set.String("args", "", "arguments for the job command")
	memoryArg := set.Uint64("mem-limit", 0, "maximum amount of memory the job command can use in bytes")
	cpuArg := set.Int("cpu-limit", 0, "maximum percentage of CPU the job command can use")
	diskIOArg := set.Int("io-limit", 0, "maximum bytes per second the job command can read and write")

	if err := parseOSArgs(set); err != nil {
		return err
	}

	s.jobCommand = *jobCommandArg
	s.args = strings.Fields(*argsArg)
	s.memoryLimit = *memoryArg
	s.cpuLimit = int32(*cpuArg)
	s.diskIOLimit = int32(*diskIOArg)

	return nil
}

func (s *StartCmd) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCtxTimeout)
	defer cancel()

	cmd := &job.Command{
		Name: s.jobCommand,
		Args: s.args,
	}

	resourceLimits := &job.Resources{
		MemoryBytes:   s.memoryLimit,
		CpuPercentage: s.cpuLimit,
		DiskIoBps:     s.diskIOLimit,
	}

	resp, err := s.client.Start(ctx, &job.StartRequest{Command: cmd, ResourceLimits: resourceLimits})

	if err != nil {
		fmt.Println(err)

		os.Exit(1)
	}

	fmt.Println(resp.Info.ID)
}
