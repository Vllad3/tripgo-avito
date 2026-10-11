package httpserver

import (
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"

	"github.com/Vllad3/tripgo-avito/internal/handler"
)

func requestValidator(spec *openapi3.T) (func(http.Handler) http.Handler, error) {
	router, err := gorillamux.NewRouter(spec)
	if err != nil {
		return nil, fmt.Errorf("build openapi router: %w", err)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route, pathParams, err := router.FindRoute(r)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			input := &openapi3filter.RequestValidationInput{
				Request:    r,
				PathParams: pathParams,
				Route:      route,
			}
			if err := openapi3filter.ValidateRequest(r.Context(), input); err != nil {
				handler.InvalidRequest(w, r, err.Error())
				return
			}

			next.ServeHTTP(w, r)
		})
	}, nil
}
