// Package cmd /*
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cbRAT",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		f, err := os.Open("./paths.cb")
		if err != nil {
			fmt.Println(err)
		}
		defer func(f *os.File) {
			err = f.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(f)

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			key := strings.Split(line, ":")[0]
			val := strings.Split(line, ":")[1]
			pathsMap[key] = val
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("...rootCmd called...")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

	// cbPaths
	rootCmd.AddCommand(cbPaths)
	cbPaths.PersistentFlags().String("scriptDir", "",
		"The location where the script will be stored.")
	cbPaths.PersistentFlags().String("modulePath", "",
		"The location of the Python module to be generated at.")
	cbPaths.PersistentFlags().String("readPath", "",
		"The file path for the openapi.generated.yaml (relative path from the pwd).")
	cbPaths.PersistentFlags().String("submodulePath", "",
		"The relative path to the submodule directory, "+
			"which stores the functions needed for the underlying REST API calls")
	cbPaths.PersistentFlags().String("confDir", "",
		"Path to the directory where the conf file are stored")
	cbPaths.PersistentFlags().String("smokeUpgradeDir", "",
		"Path to the directory where the subdirectories for the cloud providers reside. (ends with /hdbaas)")

	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	//rootCmd.PersistentFlags().StringVar(&testPath, "scriptDir", "",
	//	"Name of the Python file to generate")
	//rootCmd.PersistentFlags().StringVar(&readPath, "readPath", "",
	//	"The file path for the openapi.generated.yaml (relative path from the pwd).")
	//rootCmd.AddCommand(cbPaths)

	// cbFile
	rootCmd.AddCommand(cbFile)
	cbFile.PersistentFlags().String("nomenclature", "",
		"nomenclature for the GET endpoint of the python testcase to be scripted")
	cbFile.PersistentFlags().String("superclass", "",
		"The import string for the superclass that the current script would be using")
	cbFile.PersistentFlags().String("url", "",
		"The url that the REST API endpoint would use to send calls to")
	cbFile.PersistentFlags().String("operationId", "",
		"The operation ID specified in the YAML, for which the test has to be generated")
	cbFile.PersistentFlags().String("user", "",
		"The user that is generating the file")

	// cbModule
	rootCmd.AddCommand(cbModule)
	cbModule.PersistentFlags().String("nomenclature", "",
		"Name of the Python module to be generated")
	cbModule.PersistentFlags().String("superclass", "",
		"The superclass that the GET endpoint of the module would be using")
	cbModule.PersistentFlags().String("user", "",
		"Your name")
	cbModule.PersistentFlags().String("tag", "",
		"The tag Present in the openapi.generated.yaml"+
			"A tag string includes all the endpoints that will be automated")

	// cbSubmodule
	rootCmd.AddCommand(cbSubmodule)
	cbSubmodule.PersistentFlags().String("tag", "",
		"The tag Present in the openapi.generated.yaml for the specific endpoint")

	// cbConf
	rootCmd.AddCommand(cbConf)
	cbConf.PersistentFlags().String("tag", "",
		"The tag Present in the openapi.generated.yaml. "+
			"A tag string includes all the endpoints for which the conf file would be generated")
	cbConf.PersistentFlags().String("nomenclature", "",
		"The directory name which contain the python files for which the conf files will be generated")

	// cbSmokeUpgrade
	rootCmd.AddCommand(cbSmokeUpgrade)
	cbSmokeUpgrade.PersistentFlags().String("cloud", "",
		"The cloud provider for which the cluster for testing has to be deployed")
	cbSmokeUpgrade.PersistentFlags().String("version", "",
		"The Version of the cluster that has to be deployed")
	cbSmokeUpgrade.PersistentFlags().String("image", "",
		"The image/ami against which the test has to be run for the cluster being deployed")
	cbSmokeUpgrade.PersistentFlags().String("releaseId", "",
		"The releaseId for the image")
}
