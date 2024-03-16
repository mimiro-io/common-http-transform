package common_http_transform

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

type configUpdater struct {
	ticker *time.Ticker
	logger Logger
	config *Config
}

func (u *configUpdater) Stop(ctx context.Context) error {
	u.logger.Info("Stopping config updater")
	u.ticker.Stop()
	return nil
}

func asDuration(durationExpr string) (time.Duration, error) {
	seconds_per_unit := map[string]time.Duration{
		"s": time.Second,
		"m": time.Minute,
		"h": time.Hour,
	}
	num, err := strconv.Atoi(durationExpr[:len(durationExpr)-1])
	if err != nil {
		return 0, fmt.Errorf("invalid number in expression: %v. valid examples: 90s, 1m, 3h", durationExpr)
	}
	unitDuration, ok := seconds_per_unit[durationExpr[len(durationExpr)-1:]]
	if !ok {
		return 0, fmt.Errorf("invalid unit in expression: %v. valid examples: 90s, 1m, 3h", durationExpr)
	}
	return time.Duration(num) * unitDuration, nil
}

func newConfigUpdater(
	config *Config,
	enrichConfig func(config *Config) error,
	l Logger,
	listeners ...TransformService,
) (*configUpdater, error) {
	u := &configUpdater{logger: l}
	interval := 5 * time.Second
	if config.LayerServiceConfig.ConfigRefreshInterval != "" {
		var err error
		interval, err = asDuration(config.LayerServiceConfig.ConfigRefreshInterval)
		if err != nil {
			return nil, err
		}
	}
	u.ticker = time.NewTicker(interval)
	u.config = config

	go func() {
		for range u.ticker.C {
			u.checkForUpdates(enrichConfig, l, listeners...)
		}
	}()
	return u, nil
}

func (u *configUpdater) checkForUpdates(enrichConfig func(config *Config) error, logger Logger, listeners ...TransformService) {
	logger.Debug("checking config for updates in " + u.config.ConfigFile + ".")
	loadedConf, err := loadConfig(u.config.ConfigFile)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load config: %v", err.Error()))
		return
	}
	if enrichConfig != nil {
		err = enrichConfig(loadedConf)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to enrich config: %v", err.Error()))
			return
		}
	}
	if !u.config.equals(loadedConf) {
		logger.Info("Config changed, updating...")
		for _, listener := range listeners {
			err = listener.UpdateConfiguration(loadedConf)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to update config: %v", err.Error()))
				return
			}
		}
		// set config to the new loaded config
		u.config = loadedConf
	}
}
