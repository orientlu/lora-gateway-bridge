package cmd

import (
	"net/http"
	// pprof
	_ "net/http/pprof"

	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/brocaar/lora-gateway-bridge/internal/backend"
	"github.com/brocaar/lora-gateway-bridge/internal/config"
	"github.com/brocaar/lora-gateway-bridge/internal/forwarder"
	"github.com/brocaar/lora-gateway-bridge/internal/integration"
	"github.com/brocaar/lora-gateway-bridge/internal/metadata"
	"github.com/brocaar/lora-gateway-bridge/internal/metrics"
)

func run(cmd *cobra.Command, args []string) error {

	tasks := []func() error{
		setLogLevel,
		printStartMessage,
		setupBackend,
		setupIntegration,
		setupForwarder,
		setupMetrics,
		setupMetaData,
	}

	for _, t := range tasks {
		if err := t(); err != nil {
			log.Fatal(err)
		}
	}

	if pprofSet {
		log.WithField("url", "127.0.0.1:"+pprofPort).Warning("running in pprof model")
		go http.ListenAndServe(":"+pprofPort, nil)
	}

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	log.WithField("signal", <-sigChan).Info("signal received")
	log.Warning("shutting down server")

	return nil
}

func setLogLevel() error {
	log.SetLevel(log.Level(uint8(config.C.General.LogLevel)))
	log.SetReportCaller(logPath)
	return nil
}

func printStartMessage() error {
	log.WithFields(log.Fields{
		"version": version,
		"docs":    "https://www.loraserver.io/lora-gateway-bridge/",
	}).Info("starting LoRa Gateway Bridge")
	return nil
}

func setupBackend() error {
	if err := backend.Setup(config.C); err != nil {
		return errors.Wrap(err, "setup backend error")
	}
	return nil
}

func setupIntegration() error {
	if err := integration.Setup(config.C); err != nil {
		return errors.Wrap(err, "setup integration error")
	}
	return nil
}

func setupForwarder() error {
	if err := forwarder.Setup(config.C); err != nil {
		return errors.Wrap(err, "setup forwarder error")
	}
	return nil
}

func setupMetrics() error {
	if err := metrics.Setup(config.C); err != nil {
		return errors.Wrap(err, "setup metrics error")
	}
	return nil
}

func setupMetaData() error {
	if err := metadata.Setup(config.C); err != nil {
		return errors.Wrap(err, "setup meta-data error")
	}
	return nil
}
