package registry

import (
	"fmt"
	"go-code-samples/internal/indexer"

	lister "go-code-samples/internal/listers"

	"k8s.io/apimachinery/pkg/labels"
)

// Get all the clusters list with empty selector
func GetClusterList(clustersDirectory string) (lister.ClusterLister, error) {

	cl := lister.NewClusterLister(indexer.NewClusterIndexer(clustersDirectory))
	list, err := cl.List(labels.NewSelector())
	if err != nil {
		return nil, fmt.Errorf("cannot list clusters: %s", err)
	}
	if len(list) < 1 {
		return nil, fmt.Errorf("no matching clusters found")
	}

	return cl, nil
}
