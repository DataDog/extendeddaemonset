package metrics

import (
	"net/http"

	"k8s.io/client-go/tools/cache"
	ksmetric "k8s.io/kube-state-metrics/pkg/metric"
)

// Handler use to registry controller metrics
type Handler interface {
	RegisterStore(generators []ksmetric.FamilyGenerator, expectedType interface{}, lw cache.ListerWatcher) error
	ServeHTTP(http.ResponseWriter, *http.Request)
}
