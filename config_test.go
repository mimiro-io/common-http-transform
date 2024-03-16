package common_http_transform

import (
	"strings"
	"testing"
)

func TestConfig(t *testing.T) {
	config, err := loadConfig("./testdata")
	if err != nil {
		t.Error(err)
	}
	if config.LayerServiceConfig.ServiceName != "sample" {
		t.Error("ServiceName should be sample")
	}
	if config.ExternalSystemConfig["connection"] != "inmemory" {
		t.Error("connection should be inmemory")
	}
}

func TestConfig_AddEnvOverrides(t *testing.T) {
	t.Setenv("PORT", "8000")
	t.Setenv("CONFIG_REFRESH_INTERVAL", "60")
	t.Setenv("SERVICE_NAME", "my_service")
	t.Setenv("STATSD_ENABLED", "true")
	t.Setenv("STATSD_AGENT_ADDRESS", "localhost:8125")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "json")

	config, err := loadConfig("./testdata")
	if err != nil {
		t.Error(err)
	}
	if config.LayerServiceConfig.Port != "8000" {
		t.Error("Port should be 8000")
	}
	if config.LayerServiceConfig.ConfigRefreshInterval != "60" {
		t.Error("ConfigRefreshInterval should be 60")
	}
	if config.LayerServiceConfig.ServiceName != "my_service" {
		t.Error("ServiceName should be my_service")
	}
	if !config.LayerServiceConfig.StatsdEnabled {
		t.Error("StatsdEnabled should be true")
	}
	if config.LayerServiceConfig.StatsdAgentAddress != "localhost:8125" {
		t.Error("StatsdAgentAddress should be localhost:8125")
	}
	if config.LayerServiceConfig.LogLevel != "debug" {
		t.Error("LogLevel should be debug")
	}
	if config.LayerServiceConfig.LogFormat != "json" {
		t.Error("LogFormat should be json")
	}
}

func TestConfig_PortAsNumber(t *testing.T) {
	reader := strings.NewReader(`{ "layer_config": { "port": 8000 } }`)
	conf, err := readConfig(reader)
	if err != nil {
		t.Error(err)
	}
	if conf.LayerServiceConfig.Port != "8000" {
		t.Error("Port should be 8000")
	}
}
