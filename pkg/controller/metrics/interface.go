package metrics

import (
	"k8s.io/client-go/tools/cache"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

// Handler use to registry controller metrics.
type Handler interface {
	RegisterStore(generators []generator.FamilyGenerator, expectedType interface{}, lw cache.ListerWatcher) error
}
