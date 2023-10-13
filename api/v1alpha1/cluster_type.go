package v1alpha1

import (
	"encoding/json"
	"fmt"
	"os"

	"go-code-samples/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/yaml"
)

// Cluster describes a cluster
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// ClusterSpec is TODO
type ClusterSpec struct {
	K8sVersion        string          `json:"k8s-version,omitempty"`
	Auth              ClusterAuthSpec `json:"auth,omitempty"`
	CreationTimestamp string          `json:"creation-timestamp,omitempty"`
	Provisioner       string          `json:"provisioner,omitempty"`
	K8sDistribution   string          `json:"k8s-distro,omitempty"`
}

type ClusterAuthSpec struct {
	OIDCPRovider string `json:"oidc-provider,omitempty"`
	APIEndpoint  string `json:"api-endpoint,omitempty"`
	APICAData    string `json:"api-ca-data,omitempty"`
}

type ClusterStatus struct {
	Phase string `json:"phase,omitempty"`
}

// LoadCluster loads a Cluster from a JSON or YAML file
func LoadCluster(filename string) (*Cluster, error) {
	fileContents, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	cluster := &Cluster{}
	if utils.IsYAMLFile(filename) {
		if err := yaml.Unmarshal(fileContents, cluster); err != nil {
			return nil, fmt.Errorf("failed to parse %s as yaml: %w", filename, err)
		}
	} else if utils.IsJSONFile(filename) {
		if err := json.Unmarshal(fileContents, cluster); err != nil {
			return nil, fmt.Errorf("failed to parse %s as json: %w", filename, err)
		}
	} else {
		return nil, fmt.Errorf("unknown file type for %s", filename)
	}

	return cluster, nil
}
