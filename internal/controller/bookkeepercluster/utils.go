package bookkeepercluster

import (
	"github.com/monimesl/bookkeeper-operator/api/v1alpha1"
	"github.com/monimesl/operator-helper/k8s"
)

func getBookieSelectorLabels(c *v1alpha1.BookkeeperCluster) map[string]string {
	labels := c.GenerateWorkloadLabels(bookieComponent)
	out := make(map[string]string)
	for k, v := range labels {
		if _, found := c.Labels[k]; found {
			continue
		}
		if _, found := c.Spec.Labels[k]; found {
			continue
		}
		switch k {
		case "version":
			continue
		case k8s.LabelAppVersion:
			continue
		}
		out[k] = v
	}
	return out
}
