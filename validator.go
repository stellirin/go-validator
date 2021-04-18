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

// Initialize is a convenience function to populate the validator path cache before starting the server.
func Initialize(e *echo.Echo, doc *openapi3.Swagger) error {
	for _, r := range e.Routes() {
		ctx := e.NewContext(nil, nil)
		e.Router().Find(r.Method, r.Path, ctx)

		_, err := getRoute(r.Path, ctx.ParamNames(), doc)
		if err != nil {
			return err
		}
	}
	return nil
}

// New creates a new OpenAPI validator for Echo.
func New(doc *openapi3.Swagger, config ...Config) echo.MiddlewareFunc {
	// Set default config
	cfg := setConfig(config)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			// Don't execute the middleware if Next returns true
			if cfg.Skipper(ctx) {
				return next(ctx)
			}

			path := ctx.Path()
			req := ctx.Request()

			// get the pathItem from the cache
			// check the doc anyway, maybe we didn't run validator.Initialize()
			pathItem := pathItems.get(path)
			if pathItem == nil {
				var err error
				pathItem, err = getRoute(path, ctx.ParamNames(), doc)
				if err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, err.Error())
				}
			}

			operation := pathItem.Operations()[req.Method]
			if operation == nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("method '%s' not found on path '%s'", req.Method, req.URL.Path))
			}

			paramNames := ctx.ParamNames()
			paramValues := ctx.ParamValues()
			pathParams := make(map[string]string, len(paramValues))
			for i, v := range paramValues {
				name := paramNames[i]
				pathParams[name] = v
			}

			queryParams := ctx.QueryParams()

			validationInput := &openapi3filter.RequestValidationInput{
				Request:     req,
				PathParams:  pathParams,
				QueryParams: queryParams,
				Route: &routers.Route{
					Swagger:   doc,
					Path:      req.URL.Path,
					PathItem:  pathItem,
					Method:    req.Method,
					Operation: operation,
				},
				Options:      cfg.Options,
				ParamDecoder: cfg.ParamDecoder,
			}

			if err := openapi3filter.ValidateRequest(req.Context(), validationInput); err != nil {
				switch err.(type) {
				case *openapi3filter.RequestError:
					return echo.NewHTTPError(http.StatusBadRequest, err.Error())
				case *openapi3filter.SecurityRequirementsError:
					return echo.NewHTTPError(http.StatusForbidden, err.Error())
				default:
					return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
				}
			}

			return next(ctx)
		}
	}
}

func getRoute(path string, paramNames []string, doc *openapi3.Swagger) (*openapi3.PathItem, error) {
	// We copy because a slice is a pointer,
	// if we sort directly then ctx.ParamNames() and ctx.ParamValues() fall out of sync!
	paramCopy := make([]string, len(paramNames))
	copy(paramCopy, paramNames)

	// Sort by longest to shortest to prevent substring replacements
	if len(paramCopy) > 1 {
		sort.Slice(paramCopy, func(i, j int) bool {
			return len(paramCopy[i]) > len(paramCopy[j])
		})
	}

	apiPath := path
	for _, name := range paramCopy {
		apiPath = strings.Replace(apiPath, ":"+name, "{"+name+"}", -1)
	}

	item, exists := doc.Paths[apiPath]
	if !exists {
		return nil, fmt.Errorf("path not used in API specification: %s", path)
	}

	// path is good, cache the pathItem
	pathItems.put(path, item)

	return item, nil
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
