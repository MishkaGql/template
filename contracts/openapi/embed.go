package openapi

import (
	_ "embed"

	"github.com/getkin/kin-openapi/openapi3"
)

//go:embed trip-service.openapi.yaml
var schema []byte

func Load() (*openapi3.T, error) {
	return openapi3.NewLoader().LoadFromData(schema)
}
