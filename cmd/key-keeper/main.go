package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/fraima/key-keeper/internal/config"
	"github.com/fraima/key-keeper/internal/controller"
	"github.com/fraima/key-keeper/internal/vault"
)

var (
	Version = "undefined"
)

func main() {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level.SetLevel(zap.DebugLevel)
	logger, err := loggerConfig.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)

	var configPath string
	flag.StringVar(&configPath, "config", "", "path to config file")
	flag.Parse()

	if configPath == "" {
		zap.L().Fatal("not found config param")
	}

	cfg, err := config.Read(configPath)
	if err != nil {
		zap.L().Fatal("read configuration", zap.Error(err))
	}

	zap.L().Debug("configuration", zap.Any("config", cfg), zap.String("version", Version))

	v, err := vault.New(
		cfg.Vault,
	)
	if err != nil {
		zap.L().Fatal("init vault", zap.Error(err))
	}

	cntl := controller.New(
		v,
		cfg.Controller,
	)

	go func() {
		if err := cntl.Start(); err != nil {
			zap.L().Fatal("start", zap.Error(err))
		}
	}()

	zap.L().Info("started")

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}
