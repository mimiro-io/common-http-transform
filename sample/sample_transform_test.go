package main

import (
	"net/http"
	"strings"
	"testing"

	ct "github.com/mimiro-io/common-http-transform"
	egdm "github.com/mimiro-io/entity-graph-data-model"
)

func TestStartStopSampleTransform(t *testing.T) {
	configFile := "./config/sample_config.json"
	serviceRunner := ct.NewServiceRunner(NewSampleTransform)
	serviceRunner.WithConfigLocation(configFile)
	err := serviceRunner.Start()
	if err != nil {
		t.Error(err)
	}

	err = serviceRunner.Stop()
	if err != nil {
		t.Error(err)
	}
}

func waitForService(url string) {
	// wait for service to start.
	for {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == 200 {
			break
		}
	}
}

func TestNewSampleDataLayer(t *testing.T) {

	configFile := "./config/sample_config.json"

	serviceRunner := ct.NewServiceRunner(NewSampleTransform)
	serviceRunner.WithConfigLocation(configFile)
	err := serviceRunner.Start()
	if err != nil {
		t.Error(err)
	}

	waitForService("http://127.0.0.1:8090/health")

	// test transform, by creating entity collection with example entities
	data := `
		[
			{
				"id": "http://example.com/1",
				"props": {
					"http://example.com/name": "John Smith"
				}
			}
		]`

	reader := strings.NewReader(data)

	nsManager := egdm.NewNamespaceContext()
	parser := egdm.NewEntityParser(nsManager).WithNoContext().WithExpandURIs()
	ec, err := parser.LoadEntityCollection(reader)
	if err != nil {
		t.Error(err)
	}

	// send same data to transform service
	reader = strings.NewReader(data)
	resp, err := http.Post("http://127.0.0.1:8090/transform", "application/json", reader)
	if err != nil {
		t.Error(err)
	}

	// load a new entity collection from the response
	parser = egdm.NewEntityParser(egdm.NewNamespaceContext()).WithNoContext().WithExpandURIs()
	ec1, err := parser.LoadEntityCollection(resp.Body)
	if err != nil {
		t.Error(err)
	}

	// check that the entity collection is the same
	if ec1.Entities[0].ID != ec.Entities[0].ID {
		t.Errorf("expected entity id to be '%s', got '%s'", ec.Entities[0].ID, ec1.Entities[0].ID)
	}

	// check that the entity collection is the same
	if ec1.Entities[0].Properties["http://example.com/name"] != ec.Entities[0].Properties["http://example.com/name"] {
		t.Errorf("expected entity property to be '%s', got '%s'", ec.Entities[0].Properties["http://example.com/name"], ec1.Entities[0].Properties["http://example.com/name"])
	}

	serviceRunner.Stop()
}
