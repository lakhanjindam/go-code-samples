package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var (
	clustersDirectory string
	inputDirectory    string
	outputDirectory   string
)

// rootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "namespace-provisioner",
	Short: "Running namespace-provisioner CLI",
}

func init() {
	// generate command group
	RootCmd.AddCommand(GenerateCmd)

	addGenerateFlags(GenerateNamespaceCmd)
	GenerateCmd.AddCommand(GenerateNamespaceCmd)
}

func addGenerateFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&clustersDirectory, "clusters-dir", "c", "registry/clusters", "relative path to registry directory")
	cmd.Flags().StringVarP(&inputDirectory, "input-dir", "i", "app-provisioner-data/data/namespace", "relative path to directory containing CRD")
	cmd.Flags().StringVarP(&outputDirectory, "output-dir", "o", "output", "relative path to directory where generated manifests will be written")
}

func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
