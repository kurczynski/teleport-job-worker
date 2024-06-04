package main

import (
	"crypto/tls"
	"fmt"
	"github.com/kurczynski/teleport-job-worker/api/proto/job"
	"github.com/kurczynski/teleport-job-worker/internal/clock"
	"github.com/kurczynski/teleport-job-worker/internal/logging"
	"github.com/kurczynski/teleport-job-worker/pkg/api/serve"
	"github.com/kurczynski/teleport-job-worker/pkg/config"
	"github.com/kurczynski/teleport-job-worker/pkg/joblib/jobs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"log/slog"
	"net"
	"os"
	"time"
)

func main() {
	cfg := config.LoadServerConfig("config/server.json")

	logHandler := slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			AddSource: false,
			Level:     cfg.LogLevel,
		},
	)

	logging.Setup(logHandler)

	err := os.Setenv("SSL_CERT_DIR", cfg.Certs.CertDir)
	if err != nil {
		log.Fatal(err)
	}

	cert, err := tls.LoadX509KeyPair(cfg.Certs.CertFile, cfg.Certs.KeyFile)
	if err != nil {
		log.Fatal(err)
	}

	tlsCfg := &tls.Config{
		ClientAuth:            tls.RequireAndVerifyClientCert,
		Certificates:          []tls.Certificate{cert},
		VerifyPeerCertificate: nil,
	}

	server := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsCfg)),
	)

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))

	if err != nil {
		log.Fatal(err)
	}

	logging.Log.Info("Listening for connection", "host", cfg.Host, "port", cfg.Port)

	appClock := &clock.Application{
		Location: time.UTC,
	}

	job.RegisterJobServer(server, &serve.JobServer{
		WorkerName: cfg.WorkerName,
		Clock:      appClock,
		Jobs:       make(map[string]*jobs.Job),
	})

	if err = server.Serve(listener); err != nil {
		logging.Log.Error("Failed to serve listener", "err", err)

		os.Exit(1)
	}
}
