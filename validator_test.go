package validator_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"czechia.dev/validator"
)

func newEcho(config ...validator.Config) *echo.Echo {
	e := echo.New()

	doc, _ := openapi3.NewSwaggerLoader().LoadSwaggerFromFile("test-service.yaml")
	e.Use(validator.New(doc, config...))

	e.GET("/hello/:name", func(ctx echo.Context) error {
		name := ctx.Param("name")
		return ctx.String(http.StatusOK, fmt.Sprintf(`{"greeting": "Hello, %s!"}`, name))
	})

	e.GET("/count/:number/:currency", func(ctx echo.Context) error {
		number := ctx.Param("number")
		currency := ctx.Param("currency")
		return ctx.String(http.StatusOK, fmt.Sprintf(`{"greeting": "You got %s%s!"}`, number, currency))
	})

	validator.Initialize(e, doc)

	e.GET("/security", func(ctx echo.Context) error {
		return ctx.String(http.StatusOK, `{"greeting": "Hello!"}`)
	})

	return e
}

func Test_Echo(t *testing.T) {
	e := newEcho()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/hello/world", nil)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	require.Equal(t, http.StatusOK, res.Code)
}

func Test_Echo_Config(t *testing.T) {
	e := newEcho(validator.Config{})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/hello/world", nil)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	require.Equal(t, http.StatusOK, res.Code)
}

func Test_Echo_Bad_Path(t *testing.T) {
	e := newEcho()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/goodbye/world", nil)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	require.Equal(t, http.StatusBadRequest, res.Code)
}

func Test_Echo_Bad_Method(t *testing.T) {
	e := newEcho()

	req := httptest.NewRequest(http.MethodPost, "http://example.com/hello/world", nil)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	require.Equal(t, http.StatusBadRequest, res.Code)
}

func Test_Echo_Bad_Params(t *testing.T) {
	e := newEcho()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/count/USD/100", nil)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	require.Equal(t, http.StatusBadRequest, res.Code)
}

func Test_Echo_Bad_Security(t *testing.T) {
	e := newEcho()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/security", nil)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	require.Equal(t, http.StatusForbidden, res.Code)
}
