package main

import (
	"context"
	ct "github.com/mimiro-io/common-http-transform"
	egdm "github.com/mimiro-io/entity-graph-data-model"
)

// EnrichConfig is a function that can be used to enrich the config by reading additional files or environment variables
func EnrichConfig(config *ct.Config) error {
	config.ExternalSystemConfig["env"] = "local"
	return ct.BuildNativeSystemEnvOverrides(
		ct.Env("db_name", true),           // required env var. will fail if neiter "db_name" in json nor "DB_NAME" in env
		ct.Env("db_user", true, "dbUser"), // override jsonkey with "dbUser"
		ct.Env("db_pwd", true),
		ct.Env("db_timeout"), // optional env var. will not fail if missing in both json and ENV
	)(config)
}

/*********************************************************************************************************************/

// SampleTransform is a sample implementation of the Transform interface
type SampleTransform struct {
	config  *ct.Config
	logger  ct.Logger
	metrics ct.Metrics
}

// no shutdown required
func (dl *SampleTransform) Stop(_ context.Context) error { return nil }

func (dl *SampleTransform) Transform(ec *egdm.EntityCollection) (*egdm.EntityCollection, ct.TransformError) {
	return ec, nil
}

// NewSampleTransform is a factory function that creates a new instance of the sample transform
func NewSampleTransform(conf *ct.Config, logger ct.Logger, metrics ct.Metrics) (ct.TransformService, error) {
	sampleTransform := &SampleTransform{config: conf, logger: logger, metrics: metrics}
	return sampleTransform, nil
}

func (dl *SampleTransform) UpdateConfiguration(config *ct.Config) ct.TransformError {
	return nil
}

/*********************************************************************************************************************/
