package indexer

import (
	"os"

	v1alpha1 "go-code-samples/api/v1alpha1"
)

type ClusterIndexer interface {
	List() (ret []*v1alpha1.Cluster, err error)
}

// clusterIndexer implements the ClusterIndexer interface.
type clusterIndexer struct {
	directory string
	clusters  []*v1alpha1.Cluster
}

// NewClusterIndexer returns a new ClusterIndexer.
func NewClusterIndexer(directory string) ClusterIndexer {
	return &clusterIndexer{
		directory: directory,
	}
}

// List lists all Clusters in the indexer.
func (i *clusterIndexer) List() (ret []*v1alpha1.Cluster, err error) {
	// Return clusters if already loaded
	if len(i.clusters) > 0 {
		return i.clusters, nil
	}

	// Load clusters from disk
	clusterFiles, err := os.ReadDir(i.directory)
	if err != nil {
		return nil, err
	}
	for _, file := range clusterFiles {
		if file.IsDir() {
			continue
		}
		cluster, err := v1alpha1.LoadCluster(i.directory + "/" + file.Name())
		if err != nil {
			return nil, err
		}
		ret = append(ret, cluster)
	}

	i.clusters = ret

	return i.clusters, nil
}
