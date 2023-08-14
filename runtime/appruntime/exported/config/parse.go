package config

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/url"
)

// ParseRuntime parses the Encore runtime config.
func ParseRuntime(config, deployID string) *Runtime {
	if config == "" {
		log.Fatalln("encore runtime: fatal error: no encore runtime config provided")
	}

	// We used to support RawURLEncoding, but now we use StdEncoding.
	// Try both if StdEncoding fails.
	var (
		bytes []byte
		err   error
	)
	// nosemgrep
	if bytes, err = base64.StdEncoding.DecodeString(config); err != nil {
		bytes, err = base64.RawURLEncoding.DecodeString(config)
	}
	if err != nil {
		log.Fatalln("encore runtime: fatal error: could not decode encore runtime config:", err)
	}

	var cfg Runtime
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		log.Fatalln("encore runtime: fatal error: could not parse encore runtime config:", err)
	}

	if _, err := url.Parse(cfg.APIBaseURL); err != nil {
		log.Fatalln("encore runtime: fatal error: could not parse api base url from encore runtime config:", err)
	}

	// If the environment deploy ID is set, use that instead of the one
	// embedded in the runtime config
	if deployID != "" {
		cfg.DeployID = deployID
	}

	return &cfg
}

// ParseStatic parses the Encore static config.
func ParseStatic(config string) *Static {
	if config == "" {
		log.Fatalln("encore runtime: fatal error: no encore static config provided")
	}
	bytes, err := base64.StdEncoding.DecodeString(config)
	if err != nil {
		log.Fatalln("encore runtime: fatal error: could not decode encore static config:", err)
	}
	var cfg Static
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		log.Fatalln("encore runtime: fatal error: could not parse encore static config:", err)
	}
	return &cfg
}
