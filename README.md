# Common HTTP Transform
This is a common library to be used in creating transform services. The MIMIRO data hub allows two kinds of transforms as part of a job. The internal javascript transform and the http transform.

To utilise the HTTP transform requires an external microservice to be running that receives an array of entities and returns a new array based on some processing. That processing typically involves calls to external systems that are used to enrich data about a given entity. Examples would be a call to a weather service to update locations for current weather status. 

These HTTP transform services can be written in any language and the endpoint they provide is configurable. However, to remove all the boiler plate code around the web, config, log setup, entity parsing etc, we have created this common library. 

The common library can be imported into a standalone transform service module. The transform service then only needs to implement the TransformService interface and provide a factory function to create a new instance. The common library then provides functions to host and expose that core logic service as an HTTP service. 

Here is the entry to add to go.mod to include this common library. Note, check the release version against the latest release. 

```
github.com/mimiro-io/common-http-transform v0.1.0
```

To implement the `TransformService` interface, follow this example. This example is an identity transform, in that it returns the same as it receives.

```go
// SampleTransform is an example implementation of the Transform interface
// Note: store things like URL endpoints, or security credentials on here (obtained from config)
type SampleTransform struct {
	config  *ct.Config
	logger  ct.Logger
	metrics ct.Metrics
}

// no shutdown required
func (dl *SampleTransform) Stop(_ context.Context) error { return nil }

func (dl *SampleTransform) Transform(ec *egdm.EntityCollection) (*egdm.EntityCollection, ct.TransformError) {
	// Your transform logic goes here
	return ec, nil
}

// NewSampleTransform is a factory function that creates a new instance of the sample transform
func NewSampleTransform(conf *ct.Config, logger ct.Logger, metrics ct.Metrics) (ct.TransformService, error) {
	sampleTransform := &SampleTransform{config: conf, logger: logger, metrics: metrics}
	return sampleTransform, nil
}

// The hosting service monitors the config file for change and will notify with new config
func (dl *SampleTransform) UpdateConfiguration(config *ct.Config) ct.TransformError {
	return nil
}
```

For reference the TransformService interface is defined as follows:

```go
type Stoppable interface {
	Stop(ctx context.Context) error
}

type TransformService interface {
	Stoppable
	Transform(entityCollection *egdm.EntityCollection) (*egdm.EntityCollection, TransformError)
	UpdateConfiguration(config *Config) TransformError
}
```

Some Guidance. The NewSampleTransform function is the place to grab any config values and store then in your TransformService struct. Things such as connection strings, or urls, or credentials. 

To start up the service the following example `main.go` illustrates how to start a service. Note that your NewSampleTransform function is passed as the parameter when starting the service runner.

```go
package main

import (
	ct "github.com/mimiro-io/common-http-transform"
	"os"
)

// main function
func main() {
	args := os.Args[1:]
	configFile := args[0]
	serviceRunner := ct.NewServiceRunner(NewSampleTransform)
	serviceRunner.WithConfigLocation(configFile)
	serviceRunner.StartAndWait()
}
```

A complete sample can be found in the ./sample folder. A template project that uses this common library can be found at ...




