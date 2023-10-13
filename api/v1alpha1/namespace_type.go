package v1alpha1

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	util "go-code-samples/utils"

	"github.com/go-playground/validator/v10"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/apis/core"
)

type Namespace struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline" validate:"required"`
	metav1.ObjectMeta `json:"metadata" yaml:"metadata" validate:"required"`

	Spec NamespaceSpec `json:"spec,omitempty" yaml:"spec,omitempty" validate:"required"`
}

type NamespaceSpec struct {
	Admins          []string             `json:"admins" yaml:"admins" validate:"required"`
	ClusterSelector metav1.LabelSelector `json:"clusterSelector" yaml:"clusterSelector" validate:"required"`
	Enabled         *bool                `json:"enabled" yaml:"enabled" validate:"required"`
	Owner           string               `json:"owner" yaml:"owner" validate:"required"`
	Quotas          Quotas               `json:"quotas,omitempty" yaml:"quotas,omitempty"`
}

type Quotas struct {
	Default corev1.ResourceList `json:"default,omitempty" yaml:"default,omitempty"`
	Dev     corev1.ResourceList `json:"dev,omitempty" yaml:"dev,omitempty"`
	Stage   corev1.ResourceList `json:"stage,omitempty" yaml:"stage,omitempty"`
	Prod    corev1.ResourceList `json:"prod,omitempty" yaml:"prod,omitempty"`
}

type App struct {
	RestrictedNamespaces []string `json:"restrictedNamespaces" yaml:"restrictedNamespaces"`
}

var appConfig = &App{
	RestrictedNamespaces: []string{
		"default",
		"kube-ingress",
		"kube-node-lease",
		"kube-public",
		"kube-storageclass",
		"kube-system",
		"kube-webhooks",
	},
}

// GetAdminTeams returns a de-duplicated unique list of admin teams for a namespace
func (ns *Namespace) GetAdminTeams() []string {
	// create unique list of teams
	teamsMap := make(map[string]bool)
	for _, team := range ns.Spec.Admins {
		teamsMap[team] = true
	}
	teamsMap[ns.Spec.Owner] = true
	teams := make([]string, 0)
	for team := range teamsMap {
		teams = append(teams, team)
	}
	sort.Strings(teams)
	return teams
}

// Validate performs deeper validation on the namespace object beyond
// successful yaml parsing
func (ns *Namespace) Validate() error {

	// Validate CRD aganist the namespace struct
	validate := validator.New()
	if err := validate.Struct(ns); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		klog.Error("Validation failed for: ", ns.GetName())
		return validationErrors
	}

	if ns.Kind != "Namespace" {
		return fmt.Errorf("the kind for the yaml should be Namespace")
	}

	if reflect.DeepEqual(ns.Spec.ClusterSelector, metav1.LabelSelector{}) {
		return fmt.Errorf("clusterSelector is required")
	}
	if reflect.DeepEqual(ns.Spec.Quotas, Quotas{}) {
		return fmt.Errorf("quotas are required")
	}
	if reflect.DeepEqual(ns.Spec.Quotas.Default, corev1.ResourceList{}) {
		return fmt.Errorf("at least the default quota must be set")
	}
	// if quotas are missing for any of the environments and the default quota is not set.
	if ns.Spec.Quotas.Default == nil && (ns.Spec.Quotas.Dev == nil || ns.Spec.Quotas.Stage == nil || ns.Spec.Quotas.Prod == nil) {
		return fmt.Errorf("the default quota block must be set or define quotas for all three dev, stage and prod environments")
	}
	if err := ns.validateQuotas(); err != nil {
		return fmt.Errorf(err.Error())
	}

	// must at least select one infra boundary
	if _, ok := ns.Spec.ClusterSelector.MatchLabels["infra-boundary"]; !ok {
		// no infra-boundary selected in matchLabels, check matchExpressions too
		// since users can specify matchLabels or matchExpressions
		for _, expr := range ns.Spec.ClusterSelector.MatchExpressions {
			if expr.Key == "infra-boundary" && expr.Operator == "In" && len(expr.Values) > 0 {
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("must select atleast one infra-boundary")
		}
	}

	// Validate the labels and matchExpressions for any duplicate keys
	if err := ns.validateMatchLabelsAndExpressionsKeys(); err != nil {
		return err
	}
	// restrict the services not to use namespaces defined in config (eg: default, kube-system, kube-public,...)
	if err := hasRestrictedNamespace(ns); err != nil {
		return err
	}

	if err := validateNamespaceRegex(ns); err != nil {
		return err
	}

	return nil
}

// ValidateCRDMatchLabelsAndExpressions checks the CRD for any
// duplication of label keys used as keys in the matchExpressions block.
func (ns *Namespace) validateMatchLabelsAndExpressionsKeys() error {
	var matchedKeys []string
	if len(ns.Spec.ClusterSelector.MatchExpressions) > 0 {
		for _, expr := range ns.Spec.ClusterSelector.MatchExpressions {
			if _, ok := ns.Spec.ClusterSelector.MatchLabels[expr.Key]; ok {
				matchedKeys = append(matchedKeys, expr.Key)
			}
		}
	}
	if len(matchedKeys) != 0 {
		return fmt.Errorf("cannot have '" + strings.Join(matchedKeys, ", ") + "' as both label and key in match expressions")
	}
	return nil
}

// check if the service namespace is using any of the restricted namespaces
func hasRestrictedNamespace(ns *Namespace) error {
	// restrict the services not to use restricted namespaces (eg: default, kube-system, kube-public,...)
	if len(appConfig.RestrictedNamespaces) != 0 {
		if util.Contains(appConfig.RestrictedNamespaces, ns.GetName()) {
			return fmt.Errorf("the restricted namespaces `" + strings.Join(appConfig.RestrictedNamespaces, ", ") + "` can't be used for the service")

		}
	}
	return nil
}

// validateNamespaceName validates a namespace name according to Kubernetes
// namespace naming rules. It checks for the following:
// 1. The name must start and end with an alphanumeric character.
// 2. The name can contain hyphens and periods, but not consecutively or at the start/end.
// 3. Only lowercase letters and digits are allowed.
// 4. The length of the name must be between 1 and 63 characters.
// Returns nil if the namespace name is valid, otherwise returns an error
func validateNamespaceRegex(ns *Namespace) error {

	name := ns.GetName()
	regex := `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	if len(name) < 1 || len(name) > 63 {
		return fmt.Errorf("namespace name length must be between 1 and 63 characters")
	}
	if matched, _ := regexp.MatchString(regex, name); !matched {
		return fmt.Errorf("invalid namespace name")
	}
	if matched, _ := regexp.MatchString(`--|\.\.`, name); matched {
		return fmt.Errorf("namespace name cannot have consecutive hyphens or periods")
	}

	return nil

}

// validate quota block for each of the enviornment
func (ns *Namespace) validateQuotas() error {
	if ns.Spec.Quotas.Default != nil {
		if err := validateQuotaKeys(ns.Spec.Quotas.Default); err != nil {
			return fmt.Errorf("default resource quota: %v", err)
		}
	}
	if ns.Spec.Quotas.Dev != nil {
		if err := validateQuotaKeys(ns.Spec.Quotas.Dev); err != nil {
			return fmt.Errorf("dev resource quota: %v", err)
		}
	}
	if ns.Spec.Quotas.Stage != nil {
		if err := validateQuotaKeys(ns.Spec.Quotas.Stage); err != nil {
			return fmt.Errorf("stage resource quota: %v", err)
		}
	}
	if ns.Spec.Quotas.Prod != nil {
		if err := validateQuotaKeys(ns.Spec.Quotas.Prod); err != nil {
			return fmt.Errorf("prod resource quota: %v", err)
		}
	}
	return nil
}

// validate keys in the quota
func validateQuotaKeys(quota corev1.ResourceList) error {
	// the list of supported keys are picked from upstream core package
	// https://github.com/kubernetes/api/blob/e4c14aa9116e3ccff4d841534a400238fb4c718b/core/v1/types.go#L6375-L6409
	quotaBlockSupportedKeys := []string{
		core.ResourcePods.String(),
		core.ResourceServices.String(),
		core.ResourceServices.String(),
		core.ResourceQuotas.String(),
		core.ResourceSecrets.String(),
		core.ResourceConfigMaps.String(),
		core.ResourcePersistentVolumeClaims.String(),
		core.ResourceServicesNodePorts.String(),
		core.ResourceServicesLoadBalancers.String(),
		core.ResourceRequestsCPU.String(),
		core.ResourceRequestsMemory.String(),
		core.ResourceRequestsStorage.String(),
		core.ResourceRequestsEphemeralStorage.String(),
		core.ResourceLimitsCPU.String(),
		core.ResourceLimitsMemory.String(),
		core.ResourceLimitsEphemeralStorage.String(),
	}
	var flag bool
	var invalidKeys []string
	for key := range quota {
		if !slices.Contains(quotaBlockSupportedKeys, string(key)) {
			invalidKeys = append(invalidKeys, string(key))
			flag = true
		}
	}
	if flag {
		return fmt.Errorf(strings.Join(invalidKeys, ",") + " is/are not supported key(s) in the quota block")
	}

	return nil
}
