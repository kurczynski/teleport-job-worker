package commands

import (
	"flag"
	"github.com/kurczynski/teleport-job-worker/api/proto/job"
	"os"
	"time"
)

const (
	Start  = "start"
	Stop   = "stop"
	Query  = "query"
	Output = "output"

	DefaultCtxTimeout = 10 * time.Second
)

type Command interface {
	// ParseCLI Parses the needed arguments for the command from the CLI.
	ParseCLI(set *flag.FlagSet) error
	// Run Runs the command, i.e. interacts with the API server.
	Run()
	// SetClient Set the client to communicate with the API server.
	SetClient(job.JobClient)
}

// parseOSArgs Helper method to parse the OS args position array for the commands.
func parseOSArgs(set *flag.FlagSet) error {
	return set.Parse(os.Args[2:])
}
