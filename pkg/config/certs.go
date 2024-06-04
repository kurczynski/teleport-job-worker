package config

import (
	"crypto/tls"
	"github.com/kurczynski/teleport-job-worker/internal/logging"
	"os"
)

func ConfigureCert() (*tls.Certificate, error) {
	certDir := os.Getenv("CLI_CERT_DIR")

	if certDir == "" {
		certDir = "config/certs"
	}

	logging.Log.Info("Set certificate directory", "path", certDir)

	if err := os.Setenv("SSL_CERT_DIR", certDir); err != nil {
		return nil, err
	}

	certFile := os.Getenv("CLI_CERT_FILE")

	if certFile == "" {
		certFile = "config/certs/client-cert.pem"
	}

	logging.Log.Info("Using certificate file", "path", certFile)

	keyFile := os.Getenv("CLI_KEY_FILE")

	if keyFile == "" {
		keyFile = "config/certs/client-key.pem"
	}

	logging.Log.Info("Using certificate key", "path", keyFile)

	if cert, err := tls.LoadX509KeyPair(certFile, keyFile); err != nil {
		return nil, err
	} else {
		return &cert, nil
	}
}
