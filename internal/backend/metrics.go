package backend

import (
	"net/http"

	promhttp "github.com/prometheus/client_golang/prometheus/promhttp"
)

func MetricsHandler() http.Handler { return promhttp.Handler() }

