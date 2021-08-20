package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/karuppiah7890/go-jsonschema-generator"
	bb "github.com/runyontr/helm-schema-gen/pkg/bigbang"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var bbversion string
var path string

func init() {
	rootCmd.Flags().StringVar(&bbversion, "bb", "", "Big Bang version")
	rootCmd.Flags().StringVar(&path, "path", "", "path to bb chart")
	// flag.StringVar(&bbversion, "bb", "", "Big Bang Version")
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:           "helm schema-gen <values-yaml-file>",
	SilenceUsage:  true,
	SilenceErrors: true,
	Short:         "Helm plugin to generate json schema for values yaml",
	Long: `Helm plugin to generate json schema for values yaml

Examples:
  $ helm schema-gen values.yaml    # generate schema json
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			if bbversion != "" {
				return bb.Run(bbversion)
			}
			if path != "" {
				return bb.Run(path)
			}

			return fmt.Errorf("pass one values yaml file")
		}
		if len(args) != 1 {
			return fmt.Errorf("schema can be generated only for one values yaml at once")
		}

		valuesFilePath := args[0]
		values := make(map[string]interface{})
		valuesFileData, err := ioutil.ReadFile(valuesFilePath)
		if err != nil {
			return fmt.Errorf("error when reading file '%s': %v", valuesFilePath, err)
		}
		err = yaml.Unmarshal(valuesFileData, &values)
		s := &jsonschema.Document{}
		s.ReadDeep(&values)
		fmt.Println(s)

		return nil
	},
}

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
