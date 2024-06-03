package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/kurczynski/teleport-job-worker/api/proto/job"
	"github.com/kurczynski/teleport-job-worker/internal/logging"
	"github.com/kurczynski/teleport-job-worker/pkg/cli/commands"
	"github.com/kurczynski/teleport-job-worker/pkg/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"log/slog"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Printf("A command must be specified\n")

		os.Exit(1)
	}

	var cmd commands.Command
	var flagSet *flag.FlagSet

	switch os.Args[1] {
	case commands.Start:
		cmd = &commands.StartCmd{}
		flagSet = flag.NewFlagSet(commands.Start, flag.ExitOnError)
	case commands.Stop:
		cmd = &commands.StopCmd{}
		flagSet = flag.NewFlagSet(commands.Stop, flag.ExitOnError)
	case commands.Query:
		cmd = &commands.QueryCmd{}
		flagSet = flag.NewFlagSet(commands.Query, flag.ExitOnError)
	case commands.Output:
		cmd = &commands.OutputCmd{}
		flagSet = flag.NewFlagSet(commands.Output, flag.ExitOnError)
	default:
		log.Printf(
			"Invalid command argument; options are: %s, %s, %s, %s\n",
			commands.Start, commands.Stop, commands.Query, commands.Output,
		)

		os.Exit(1)
	}

	var logArg string
	var portArg int
	var hostArg string

	flagSet.StringVar(&logArg, "log-level", "info", "set the log level; one of: debug, info, warn, error")
	flagSet.IntVar(&portArg, "port", 8443, "port to use when connecting to the server")
	flagSet.StringVar(&hostArg, "host", "localhost", "host to connect to")

	if err := cmd.ParseCLI(flagSet); err != nil {
		log.Println(err)

		os.Exit(1)
	}

	logLevel, err := parseLogLevel(logArg)

	if err != nil {
		log.Println(err)

		os.Exit(1)
	}

	logHandler := slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			AddSource: false,
			Level:     logLevel,
		},
	)

	logging.Setup(logHandler)

	cert, err := config.ConfigureCert()

	if err != nil {
		logging.Log.Error("Failed to configure certificate", "err", err)

		os.Exit(2)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{*cert},
		MinVersion:   tls.VersionTLS13,
	}

	logging.Log.Info("Connecting to server", "host", hostArg, "port", portArg)

	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:%d", hostArg, portArg),
		grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)),
	)

	if err != nil {
		logging.Log.Error("Failed to create client", "err", err)

		os.Exit(1)
	}

	defer conn.Close()

	client := job.NewJobClient(conn)

	logging.Log.Info("Successfully connected to server", "host", hostArg, "port", portArg)

	cmd.SetClient(client)

	cmd.Run()
}

func parseLogLevel(levelName string) (slog.Level, error) {
	var level slog.Level
	var err = level.UnmarshalText([]byte(levelName))

	return level, err
}
