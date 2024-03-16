package common_http_transform

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func (serviceRunner *ServiceRunner) WithEnrichConfig(enrichConfig func(config *Config) error) *ServiceRunner {
	serviceRunner.enrichConfig = enrichConfig
	return serviceRunner
}

func (serviceRunner *ServiceRunner) WithConfigLocation(configLocation string) *ServiceRunner {
	serviceRunner.configLocation = configLocation
	return serviceRunner
}

func NewServiceRunner(newTransformService func(config *Config, logger Logger, metrics Metrics) (TransformService, error)) *ServiceRunner {
	runner := &ServiceRunner{}
	runner.createService = newTransformService
	return runner
}

func (serviceRunner *ServiceRunner) configure() {
	if serviceRunner.configLocation == "" {
		configPath, found := os.LookupEnv("DATALAYER_CONFIG_PATH")
		if found {
			serviceRunner.configLocation = configPath
		} else {
			serviceRunner.configLocation = "./config"
		}
	}

	config, err := loadConfig(serviceRunner.configLocation)
	if err != nil {
		panic(err)
	}

	// enrich config specific for layer
	if serviceRunner.enrichConfig != nil {
		err = serviceRunner.enrichConfig(config)
		if err != nil {
			panic(err)
		}
	}

	// initialise logger
	logger := NewLogger(
		config.LayerServiceConfig.ServiceName,
		config.LayerServiceConfig.LogFormat,
		config.LayerServiceConfig.LogLevel,
	)
	serviceRunner.logger = logger

	metrics, err := newMetrics(config)
	if err != nil {
		panic(err)
	}

	serviceRunner.transformService, err = serviceRunner.createService(config, logger, metrics)
	if err != nil {
		panic(err)
	}

	// create and start config updater
	serviceRunner.configUpdater, err = newConfigUpdater(config, serviceRunner.enrichConfig, logger, serviceRunner.transformService)
	if err != nil {
		panic(err)
	}

	// create web service hook up with the service core
	serviceRunner.webService, err = newTransformService(config, logger, metrics, serviceRunner.transformService)
	if err != nil {
		panic(err)
	}

	serviceRunner.stoppable = append(
		serviceRunner.stoppable,
		serviceRunner.transformService,
		serviceRunner.configUpdater,
		serviceRunner.webService)
}

type ServiceRunner struct {
	logger           Logger
	enrichConfig     func(config *Config) error
	webService       *transformWebService
	configUpdater    *configUpdater
	createService    func(config *Config, logger Logger, metrics Metrics) (TransformService, error)
	configLocation   string
	transformService TransformService
	stoppable        []Stoppable
}

func (serviceRunner *ServiceRunner) TransformService() TransformService {
	return serviceRunner.transformService
}

func (serviceRunner *ServiceRunner) Start() error {
	// configure the service
	serviceRunner.configure()

	// start the service
	err := serviceRunner.webService.Start()
	if err != nil {
		return err
	}

	return nil
}

func (serviceRunner *ServiceRunner) StartAndWait() {
	// configure the service
	serviceRunner.configure()

	// start the service
	err := serviceRunner.webService.Start()
	if err != nil {
		panic(err)
	}

	// and wait for ctrl-c
	serviceRunner.andWait()
}

func (serviceRunner *ServiceRunner) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, stoppable := range serviceRunner.stoppable {
		err := stoppable.Stop(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (serviceRunner *ServiceRunner) andWait() {
	// handle shutdown, this call blocks and keeps the application running
	waitForStop(serviceRunner.logger, serviceRunner.stoppable...)
}

//	 waitForStop listens for SIGINT (Ctrl+C) and SIGTERM (graceful docker stop).
//		It accepts a list of stoppables that will be stopped when a signal is received.
func waitForStop(logger Logger, stoppable ...Stoppable) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	logger.Info("Data Layer stopping")

	shutdownCtx := context.Background()
	wg := sync.WaitGroup{}
	for _, s := range stoppable {
		s := s
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.Stop(shutdownCtx)
			if err != nil {
				logger.Error("Stopping Data Layer failed: %+v", err)
				os.Exit(2)
			}
		}()
	}
	wg.Wait()
	logger.Info("Data Layer stopped")
	os.Exit(0)
}
