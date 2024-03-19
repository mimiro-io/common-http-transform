package main

import (
	"context"
	ct "github.com/mimiro-io/common-http-transform"
	egdm "github.com/mimiro-io/entity-graph-data-model"
	"sync"
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
	result, err := processEntitiesConcurrently(ec.Entities, 10, processEntity)
	if err != nil {
		return nil, ct.Err(err, 1)
	}
	return &egdm.EntityCollection{Entities: result}, nil
}

type Result struct {
	Index  int
	Entity *egdm.Entity
	Error  error
}

func processEntitiesConcurrently(entities []*egdm.Entity, concurrency int, handler func(entity *egdm.Entity) (*egdm.Entity, error)) ([]*egdm.Entity, error) {
	var wg sync.WaitGroup
	results := make([]*egdm.Entity, len(entities))
	resultsChan := make(chan Result, len(entities))

	// Number of goroutines to use
	itemsPerGoroutine := (len(entities) + concurrency - 1) / concurrency

	for i := 0; i < concurrency; i++ {
		start := i * itemsPerGoroutine
		end := start + itemsPerGoroutine
		if end > len(entities) {
			end = len(entities)
		}

		wg.Add(1)
		go func(start, end, index int) {
			defer wg.Done()
			for j := start; j < end; j++ {
				result, err := handler(entities[j])
				resultsChan <- Result{Index: j, Entity: result, Error: err}
			}
		}(start, end, i)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for result := range resultsChan {
		if result.Error != nil {
			return nil, result.Error
		}
		results[result.Index] = result.Entity
	}

	return results, nil
}

func processEntity(entity *egdm.Entity) (*egdm.Entity, error) {
	return entity, nil
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
