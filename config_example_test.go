package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/iver-wharf/wharf-core/pkg/config"
)

func buildTestConfig(configYAML string) (Config, error) {
	var builder = config.NewBuilder(DefaultConfig)
	builder.AddEnvironmentVariables("WHARF")
	builder.AddConfigYAML(strings.NewReader(configYAML))
	var config Config
	err := builder.Unmarshal(&config)
	return config, err
}

func ExampleConfig() {
	var configYAML = `
http:
  cors:
    allowAllOrigins: true

api:
  url: https://wharf.example.org/api
`

	// Prefix of WHARF_ must be prepended to all environment variables
	os.Setenv("WHARF_HTTP_BINDADDRESS", "0.0.0.0:8123")

	var config, err = buildTestConfig(configYAML)
	if err != nil {
		fmt.Println("Error reading config:", err)
		return
	}

	fmt.Println("Allow any CORS?", config.HTTP.CORS.AllowAllOrigins)
	fmt.Println("HTTP bind address:", config.HTTP.BindAddress)
	fmt.Println("Wharf API URL:", config.API.URL)

	// Output:
	// Allow any CORS? true
	// HTTP bind address: 0.0.0.0:8123
	// Wharf API URL: https://wharf.example.org/api
}
