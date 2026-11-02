package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewMetricsHandler() (http.Handler, error) {
	return promhttp.Handler(), nil
}
