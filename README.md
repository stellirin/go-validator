# Validator: OpenAPI 3.x validation middleware for Echo

[![reference](https://pkg.go.dev/badge/czechia.dev/validator.svg)](https://pkg.go.dev/czechia.dev/validator)
[![report](https://goreportcard.com/badge/czechia.dev/validator)](https://goreportcard.com/report/czechia.dev/validator)
[![tests](https://github.com/stellirin/go-validator/workflows/Go/badge.svg)](https://github.com/stellirin/go-validator/actions?query=workflow%3AGo)
[![coverage](https://codecov.io/gh/stellirin/go-validator/branch/main/graph/badge.svg?token=Q8irv4HHtY)](https://codecov.io/gh/stellirin/go-validator)

A simple package to validate incoming requests against an OpenAPI specification for Echo.

## ⚙️ Installation

```sh
go get -u czechia.dev/validator
```

## 📝 Usage

Other [OpenAPI validators](https://github.com/deepmap/oapi-codegen/blob/v1.6.0/pkg/middleware/oapi_validate.go#L70) use a router from the [`kin-openapi`](https://github.com/getkin/kin-openapi) package within the validator itself. This means a request is 'routed' twice, once by Echo, and then again by the validator.

`czechia.cz/validator` takes advantage of the fact that Echo has already routed valid requests, so the handler path, parameters, etc. are all available in the Echo context. We simply look for the corresponding path in the OpenAPI specification and validate against it directly.

The validator maintains a cached list of Echo Routes and their associated OpenAPI paths to speed up validation. You can prepopulate the path cache by setting the Echo Route Name as the OpenAPI path and call the `validator.Initialize()` function.

If the OpenAPI path is not found in the cache at runtime then the validator searches for the path according to the route's parameter names. This means your Echo parameters must have the same names as the OpenAPI parameters.

Example: if the Echo handler path is `/hello/:name` then we look for the `/hello/{name}` OpenAPI path.

Ultimtely, if no OpenAPI path is found then the response is always an error.

## 👀 Example

```go
package main

import (
	"fmt"
	"net/http"

	"czechia.dev/validator"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()
	spec, _ := openapi3.NewLoader().LoadFromFile("test-service.yaml")
	e.Use(validator.New(spec))

	e.GET("/hello/:name", func(ctx echo.Context) error {
		return ctx.String(http.StatusOK, fmt.Sprintf(`{"greeting": "Hello, %s!"}`, ctx.Param("name")))
	}).Name = "/hello/{name}"

	validator.Initialize(e, spec)

	e.Start(":8080")
}
```
