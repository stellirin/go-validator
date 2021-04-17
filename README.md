# Validator: OpenAPI 3.x validation middleware for Echo

[![codecov](https://codecov.io/gh/stellirin/go-validator/branch/main/graph/badge.svg?token=Q8irv4HHtY)](https://codecov.io/gh/stellirin/go-validator)
[![Test Action Status](https://github.com/stellirin/go-validator/workflows/Go/badge.svg)](https://github.com/stellirin/go-validator/actions?query=workflow%3AGo)

A simple package to validate incoming requests against an OpenAPI specification for Echo.

## ⚙️ Installation

```sh
go get -u czechia.dev/validator
```

## 📝 Usage

Other validators use an OpenAPI router from the `kin-openapi` package within the validator itself. This means a request is 'routed' twice, once by Echo, and the again by the validator.

*This* validator takes advantage of the fact that Echo has already routed the request, and the handler path, parameters, etc. are all available in the Echo context. We simply look for the corresponding path in the OpenAPI specification and validate it directly.

If the Echo handler path is `/hello/:name` then we look for the `/hello/{name}` OpenAPI path. This means your Echo parameters must have the same names as the OpenAPI parameters.

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
	doc, _ := openapi3.NewSwaggerLoader().LoadSwaggerFromFile("test-service.yaml")
	e.Use(validator.New(doc))

	e.GET("/hello/:name", func(ctx echo.Context) error {
		name := ctx.Param("name")
		return ctx.String(http.StatusOK, fmt.Sprintf(`{"greeting": "Hello, %s!"}`, name))
	})

	e.GET("/count/:number/:currency", func(ctx echo.Context) error {
		number := ctx.Param("number")
		currency := ctx.Param("currency")
		return ctx.String(http.StatusOK, fmt.Sprintf(`{"greeting": "You got %s%s!"}`, number, currency))
	})

	e.GET("/security", func(ctx echo.Context) error {
		return ctx.String(http.StatusOK, `{"greeting": "Hello!"}`)
	})

	e.Start(":8080")
}
```
