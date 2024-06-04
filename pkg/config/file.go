package config

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
)

type ServerConfig struct {
	Certs      Certs      `json:"certs"`
	Host       string     `json:"host"`
	LogLevel   slog.Level `json:"logLevel"`
	Port       int        `json:"port"`
	WorkerName string     `json:"workerName"`
}

type Certs struct {
	CertDir  string `json:"certDir"`
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
}

func LoadServerConfig(fname string) *ServerConfig {
	configFile, err := os.ReadFile(fname)

	if err != nil {
		log.Fatal(err)
	}

	config := &ServerConfig{}

	err = json.Unmarshal(configFile, config)

	return config
}
