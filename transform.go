package common_http_transform

import (
	"context"

	egdm "github.com/mimiro-io/entity-graph-data-model"
)

type Stoppable interface {
	Stop(ctx context.Context) error
}

type TransformServiceFactory interface {
	Build(config *Config, logger Logger, metrics Metrics) (TransformService, error)
}

type TransformService interface {
	Stoppable
	Transform(entityCollection *egdm.EntityCollection) (*egdm.EntityCollection, TransformError)
	UpdateConfiguration(config *Config) TransformError
}
