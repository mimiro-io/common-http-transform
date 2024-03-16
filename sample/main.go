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
	serviceRunner.WithEnrichConfig(EnrichConfig)
	serviceRunner.StartAndWait()
}
