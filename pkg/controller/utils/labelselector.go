package utils

import (
	"github.com/go-logr/logr"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

// ConvertLabelSelector converts a "k8s.io/apimachinery/pkg/apis/meta/v1".LabelSelector as found in manifests spec section into a "k8s.io/apimachinery/pkg/labels".Selector to be used to filter list operations.
func ConvertLabelSelector(logger logr.Logger, inSelector *metav1.LabelSelector) (outSelector labels.Selector, err error) {
	outSelector = labels.NewSelector()
	if inSelector != nil {
		for key, value := range inSelector.MatchLabels {
			req, err2 := labels.NewRequirement(key, selection.In, []string{value})
			if err2 != nil {
				logger.Error(err, "NewRequirement")
				err = err2
				continue
			}
			outSelector = outSelector.Add(*req)
		}

		for _, expr := range inSelector.MatchExpressions {
			var op selection.Operator
			switch expr.Operator {
			case metav1.LabelSelectorOpIn:
				op = selection.In
			case metav1.LabelSelectorOpNotIn:
				op = selection.NotIn
			case metav1.LabelSelectorOpExists:
				op = selection.Exists
			case metav1.LabelSelectorOpDoesNotExist:
				op = selection.DoesNotExist
			default:
				logger.Info("Invalid Operator:", expr.Operator)
				continue
			}
			req, err2 := labels.NewRequirement(expr.Key, op, expr.Values)
			if err2 != nil {
				logger.Error(err, "NewRequirement")
				err = err2
				continue
			}
			outSelector = outSelector.Add(*req)
		}
	}
	return outSelector, err
}
