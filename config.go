package common_http_transform

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

type Config struct {
	ConfigFile           string               // set by service runner
	ExternalSystemConfig ExternalSystemConfig `json:"external_config"`
	LayerServiceConfig   *LayerServiceConfig  `json:"layer_config"`
}

type ExternalSystemConfig map[string]any

type LayerServiceConfig struct {
	Custom                map[string]any `json:"custom"`
	ServiceName           string         `json:"service_name"`
	Port                  json.Number    `json:"port"`
	ConfigRefreshInterval string         `json:"config_refresh_interval"`
	LogLevel              string         `json:"log_level"`
	LogFormat             string         `json:"log_format"`
	StatsdAgentAddress    string         `json:"statsd_agent_address"`
	StatsdEnabled         bool           `json:"statsd_enabled"`
}

/******************************************************************************/
type EnvOverride struct {
	EnvVar   string
	ConfKey  string
	Required bool
}

// Env function to conveniently construct EnvOverride instances
func Env(key string, specs ...any) EnvOverride {
	e := EnvOverride{EnvVar: key}
	for _, spec := range specs {
		switch v := spec.(type) {
		case bool:
			e.Required = v
		case string:
			e.ConfKey = v
		}
	}
	return e
}

// BuildNativeSystemEnvOverrides can be plugged into `WithEnrichConfig`
//
//	it takes a variadic parameter list, each of which declares an environment variable
//	that the layer will try to look up at start, and add to system_config.
func BuildNativeSystemEnvOverrides(envOverrides ...EnvOverride) func(config *Config) error {
	return func(config *Config) error {
		for _, envOverride := range envOverrides {
			upper := strings.ToUpper(envOverride.EnvVar)
			key := strings.ToLower(envOverride.EnvVar)
			if envOverride.ConfKey != "" {
				key = envOverride.ConfKey
			}
			if v, ok := os.LookupEnv(upper); ok {
				config.ExternalSystemConfig[key] = v
			} else if envOverride.Required {
				_, confFound := config.ExternalSystemConfig[key]
				if !confFound {
					return fmt.Errorf("required system_config variable %s not found in config nor LookupEnv(%s)", key, upper)
				}
			}

		}
		return nil
	}
}

func (c *Config) equals(conf *Config) bool {
	return reflect.DeepEqual(c, conf)
}

func newConfig() *Config {
	return &Config{}
}

func readConfig(data io.Reader) (*Config, error) {
	config := newConfig()
	s, err := io.ReadAll(data)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(s, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func loadConfig(configPath string) (*Config, error) {
	reader, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	config, err := readConfig(reader)
	if err != nil {
		return nil, err
	}

	config.ConfigFile = configPath

	return config, nil
}

func addEnvOverrides(c *Config) {
	val, found := os.LookupEnv("PORT")
	if found {
		c.LayerServiceConfig.Port = json.Number(val)
	}

	val, found = os.LookupEnv("CONFIG_REFRESH_INTERVAL")
	if found {
		c.LayerServiceConfig.ConfigRefreshInterval = val
	}

	val, found = os.LookupEnv("SERVICE_NAME")
	if found {
		c.LayerServiceConfig.ServiceName = val
	}

	val, found = os.LookupEnv("STATSD_ENABLED")
	if found {
		c.LayerServiceConfig.StatsdEnabled = val == "true"
	}

	val, found = os.LookupEnv("STATSD_AGENT_ADDRESS")
	if found {
		c.LayerServiceConfig.StatsdAgentAddress = val
	}

	val, found = os.LookupEnv("LOG_LEVEL")
	if found {
		c.LayerServiceConfig.LogLevel = val
	}

	val, found = os.LookupEnv("LOG_FORMAT")
	if found {
		c.LayerServiceConfig.LogFormat = val
	}
}
