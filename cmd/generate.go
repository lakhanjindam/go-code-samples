package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-code-samples/internal/registry"

	"go-code-samples/api/v1alpha1"

	"github.com/spf13/cobra"
	"github.com/twilio-internal/otk-app-provisioner/controllers/namespace"
	"k8s.io/klog/v2"
	k8syaml "sigs.k8s.io/yaml"
)

var GenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate kubernetes manifests",
	Run:   GenerateCmdFunc,
}

var GenerateNamespaceCmd = &cobra.Command{
	Use:   "namespace",
	Short: "Generate namespace kubernetes manifests",
	Run:   GenerateNamespaceCmdFunc,
}

// generate command information
func GenerateCmdFunc(cmd *cobra.Command, args []string) {
	fmt.Println("Generate kubernetes manifests")
}

// generate namespace command execution
func GenerateNamespaceCmdFunc(cmd *cobra.Command, args []string) {
	fmt.Println("Generate namespace kubernetes manifests")

	if _, err := registry.GetClusterList(clustersDirectory); err != nil {
		klog.Fatalf("cannot list clusters: %s", err)
	}

	// get all input files
	err := filepath.Walk(inputDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), "yaml") || strings.HasSuffix(info.Name(), "yml") {
			ns := &v1alpha1.Namespace{}
			fileContents, err := os.ReadFile(path)
			if err != nil {
				klog.Fatal("cannot read file %s: %s", path, err)
			}
			if err = k8syaml.Unmarshal(fileContents, ns); err != nil {
				klog.Fatal("cannot parse file %s: %s", path, err)
			}
			if err = ns.Validate(); err != nil {
				klog.Fatal("cannot validate namespace %s.yaml for service %s: %s", ns.GetName(), err)
			}
			if *ns.Spec.Enabled {
				// reconcile function will select target clusters based on the clusterSelector and then perform various actions
				//1. Create a namespace k8s resource
				//2. create a rbac role for the namespace
				//3. create a rbac rolebinding for the namespace
				//4. create a quota for the namespace
				// All of the files will be created for above steps and output to a cluster specific directory
				return namespace.Reconcile(ns, clustersDirectory, outputDirectory)
			} else {
				fmt.Printf("Enabled: %t for config %s\n", *ns.Spec.Enabled, path)
			}

		}
		return nil
	})
	if err != nil {
		klog.Fatal("failed to reconcile namespace configuration for app: %s", err)
	}
}
