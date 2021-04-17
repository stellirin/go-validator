package validator

import (
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4/middleware"
)

// Config defines the Validator config for the middleware.
type Config struct {

	// Skipper defines a function to skip this middleware when returned true.
	// This field is used only by Echo.
	//
	// Optional. Default: nil
	Skipper middleware.Skipper

	Options      *openapi3filter.Options
	ParamDecoder openapi3filter.ContentParameterDecoder
}

// defaultConfig is the default config
var defaultConfig = Config{
	Skipper: middleware.DefaultSkipper,
}

// Helper function to set default values
func setConfig(config []Config) Config {
	// Return default config if nothing provided
	if len(config) <= 0 {
		cfg := defaultConfig
		return cfg
	}

	// Override default config
	cfg := config[0]

	// Set default values
	if cfg.Skipper == nil {
		cfg.Skipper = defaultConfig.Skipper
	}
	if cfg.Options == nil {
		cfg.Options = defaultConfig.Options
	}
	if cfg.ParamDecoder == nil {
		cfg.ParamDecoder = defaultConfig.ParamDecoder
	}

	return cfg
}
