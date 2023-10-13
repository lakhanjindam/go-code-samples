package listers

import (
	clusterindexer "go-code-samples/internal/indexer"

	v1alpha1 "go-code-samples/api/v1alpha1"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
)

// ClusterLister helps list Clusters.
type ClusterLister interface {
	// List lists all Pods in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Cluster, err error)
}

// clusterLister implements the ClusterLister interface.
type clusterLister struct {
	indexer clusterindexer.ClusterIndexer
}

// NewClusterLister returns a new ClusterLister.
func NewClusterLister(indexer clusterindexer.ClusterIndexer) ClusterLister {
	return &clusterLister{indexer: indexer}
}

// List lists all Clusters in the indexer with optional label selector filtering.
func (s *clusterLister) List(selector labels.Selector) (ret []*v1alpha1.Cluster, err error) {
	clusters, err := s.indexer.List()
	if err != nil {
		return nil, err
	}
	if selector.Empty() {
		// User wants all clusters
		return clusters, nil
	}
	for _, m := range clusters {
		metadata, err := meta.Accessor(m)
		if err != nil {
			return nil, err
		}
		if selector.Matches(labels.Set(metadata.GetLabels())) {
			ret = append(ret, m)
		}
	}

	return ret, nil
}
