package commands

import (
	"flag"
	"github.com/kurczynski/teleport-job-worker/api/proto/job"
)

// Command constants that can be used in the CLI.
const (
	Start  = "start"
	Stop   = "stop"
	Query  = "query"
	Output = "output"
)

type Command interface {
	// ParseCLI Parses the needed arguments for the command from the CLI.
	ParseCLI(set *flag.FlagSet) error
	// Run Runs the command, i.e. interacts with the API server.
	Run()
	// SetClient Set the client to communicate with the API server.
	SetClient(job.JobClient)
}
