package validator

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/labstack/echo/v4"
)

// cache of openapi3.PathItem using echo.Route.Path as the key
var pathItems *pathMap

func init() {
	pathItems = &pathMap{
		mutex: sync.RWMutex{},
		paths: make(map[string]*openapi3.PathItem),
	}
}

// Initialize populates the validator path cache before starting the server.
//
// You must set the route Name to the OpenAPI path, e.g.:
//	e.GET("/hello/:name", helloHandler).Name = "/hello/{name}"
func Initialize(e *echo.Echo, s *openapi3.Swagger) {
	for _, r := range e.Routes() {
		if item := s.Paths[r.Name]; item != nil {
			pathItems.put(r.Path, item)
		}
	}
}

// New creates a new OpenAPI validator for Echo.
func New(swagger *openapi3.Swagger, config ...Config) echo.MiddlewareFunc {
	// Set default config
	cfg := setConfig(config)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			// Don't execute the middleware if Next returns true
			if cfg.Skipper(ctx) {
				return next(ctx)
			}

			req := ctx.Request()

			pathItem := getRoute(ctx, swagger)
			if pathItem == nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("path not valid: %s", req.URL.Path))
			}

			operation := pathItem.Operations()[req.Method]
			if operation == nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("method '%s' not valid on path '%s'", req.Method, req.URL.Path))
			}

			paramNames := ctx.ParamNames()
			paramValues := ctx.ParamValues()
			pathParams := make(map[string]string, len(paramValues))
			for i, v := range paramValues {
				name := paramNames[i]
				pathParams[name] = v
			}

			queryParams := ctx.QueryParams()

			input := &openapi3filter.RequestValidationInput{
				Request:     req,
				PathParams:  pathParams,
				QueryParams: queryParams,
				Route: &routers.Route{
					Swagger:   swagger,
					Path:      req.URL.Path,
					PathItem:  pathItem,
					Method:    req.Method,
					Operation: operation,
				},
				Options:      cfg.Options,
				ParamDecoder: cfg.ParamDecoder,
			}

			if err := openapi3filter.ValidateRequest(req.Context(), input); err != nil {
				switch validateErr := err.(type) {
				case *openapi3filter.RequestError:
					switch requestErr := validateErr.Err.(type) {
					case *openapi3filter.ParseError:
						return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf(
							"cannot parse parameter '%s' in %s, got '%v' but %s",
							validateErr.Parameter.Name,
							validateErr.Parameter.In,
							requestErr.Value,
							requestErr.Reason,
						))
					case *openapi3.SchemaError:
						return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf(
							"cannot parse parameter '%s' in %s, got '%v' but %s",
							validateErr.Parameter.Name,
							validateErr.Parameter.In,
							requestErr.Value,
							requestErr.Reason,
						))
					default:
						return echo.NewHTTPError(http.StatusBadRequest, validateErr.Err.Error())
					}
				case *openapi3filter.SecurityRequirementsError:
					return echo.NewHTTPError(http.StatusForbidden, validateErr.Error())
				default:
					return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
				}
			}

			return next(ctx)
		}
	}
}

func getRoute(ctx echo.Context, swagger *openapi3.Swagger) *openapi3.PathItem {
	// Path is in the cache?
	if pathItem := pathItems.get(ctx.Path()); pathItem != nil {
		return pathItem
	}

	// don't sort ctx.ParamNames() directly else ctx.ParamNames() and ctx.ParamValues() fall out of sync!
	paramNames := make([]string, len(ctx.ParamNames()))
	copy(paramNames, ctx.ParamNames())

	// Sort by longest to shortest to prevent substring replacements
	if len(paramNames) > 1 {
		sort.Slice(paramNames, func(i, j int) bool {
			return len(paramNames[i]) > len(paramNames[j])
		})
	}

	apiPath := ctx.Path()
	for _, name := range paramNames {
		// OpenAPI parameters must appear only once in the path
		apiPath = strings.Replace(apiPath, ":"+name, "{"+name+"}", 1)
	}

	item, exists := swagger.Paths[apiPath]
	if exists {
		// path is good, cache it!
		pathItems.put(ctx.Path(), item)
	}

	return item
}

type pathMap struct {
	mutex sync.RWMutex
	paths map[string]*openapi3.PathItem
}

func (pm *pathMap) put(key string, value *openapi3.PathItem) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.paths[key] = value
}

func (pm *pathMap) get(key string) *openapi3.PathItem {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	return pm.paths[key]
}
