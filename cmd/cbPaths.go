package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var pathsMap = map[string]string{
	"modulePath":      "",
	"scriptDir":       "",
	"readPath":        "",
	"submodulePath":   "",
	"confDir":         "",
	"smokeUpgradeDir": "",
}

// cbPaths represents the cbPaths command
var cbPaths = &cobra.Command{
	Use:   "cbPaths",
	Short: "Specify the path where the conf file would be created.",
	Long: `cbPaths:
The command can be used to set the path of the TAF directory relative to the current directory's path, 
but the path should also include the subdirectory inside which the automated scripts would be placed.
And it can also be used to set the path of the directory in which the openapi.generated.yaml is stored in, 
it will be used to read the specs of the API (to be scripted) from the openapi.generated.yaml file. 

Usage example:
cbReadPath --scriptDir "../foo/bar/TAF/pytests/Capella/RestAPIv4/" --readPath "../foo/bar/openapi.generated.yaml"`,
	PreRun: func(cmd *cobra.Command, args []string) {
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			if !flag.Changed {
				return
			}
			fmt.Println(flag.Name)
			pathsMap[flag.Name] = validateAndAddPathForFlag(flag.Name, flag.Value.String())
		})
	},
	Run: func(cmd *cobra.Command, args []string) {
		for key, value := range pathsMap {
			fmt.Println(key, ":", value)
		}
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		f, err := os.Create("./paths.cb")
		if err != nil {
			fmt.Println(err)
		}
		defer func(f *os.File) {
			err = f.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(f)

		var stdOutPaths string
		for key, val := range pathsMap {
			stdOutPaths += key + ":" + val + "\n"
		}
		_, err = f.WriteString(stdOutPaths)
		if err != nil {
			fmt.Println(err)
		}
	},
}

func validateAndAddPathForFlag(name string, val string) string {
	switch name {

	case "scriptDir":
		if val == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter a valid directory path that currently exists for a module to be instantiated " +
				"inside it/files to be created inside : ")
			val, _ = reader.ReadString('\n')
			val = strings.TrimSpace(val)
		}

		// Open the directory
		dir, err := os.Open(val)
		if val == "" || err != nil {
			fmt.Println("Error opening directory : ", err)
			fmt.Printf("There was an error opening the directory : " + val +
				" please verify if the specified directory is correct or not...!!!")
			return ""
		}
		defer func(dir *os.File) {
			err = dir.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(dir)

	case "modulePath":
		if val == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter a valid directory path that currently contains a python module, " +
				"for a file to be instantiated inside it : ")
			val, _ = reader.ReadString('\n')
			val = strings.TrimSpace(val)
		}

		// Open the directory
		dir, err := os.Open(val)
		if val == "" || err != nil {
			fmt.Println("Error opening directory : ", err)
			fmt.Printf("There was an error opening the directory : " + val +
				" please verify if the specified directory is correct or not...!!!")
			return ""
		}
		defer func(dir *os.File) {
			err = dir.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(dir)

	case "readPath":
		if val == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter a valid directory path that currently holds the openapi.generated.yaml inside it : ")
			val, _ = reader.ReadString('\n')
			val = strings.TrimSpace(val)
		}
		if val == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter the relative file path for the openapi.generated.yaml : ")
			val, _ = reader.ReadString('\n')
			val = strings.TrimSpace(val)
		}

		// Process the readPath provided and see if the directory contains openapi.generated.yaml or not.
		lastDir := strings.Split(val, "/")[len(strings.Split(val, "/"))-1]
		if lastDir == "openapi.generated.yaml" {
			val = val[:len(val)-len("openapi.generated.yaml")-1]
		} else if lastDir == "" {
			val = val[:len(val)-1]
		}

		// Open the directory
		dir, err := os.Open(val)
		if err != nil {
			fmt.Println("Error opening directory : ", err)
			fmt.Printf("There was an error opening the directory : `%s` , "+
				"please verify if the specified directory is correct or not...!!!", val)
			return ""
		}
		defer func(dir *os.File) {
			err = dir.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(dir)

		// Read the directory
		files, err := dir.Readdir(-1)
		if err != nil {
			fmt.Println("Error reading directory -", val, " , Error : ", err)
			return ""
		}

		// Find openapi.generated.yaml -
		foundYaml := false
		for _, file := range files {
			if file.Name() == "openapi.generated.yaml" {
				foundYaml = true
				break
			}
		}
		if val == "" || !foundYaml {
			fmt.Println("Please provide a valid file path from the pwd for the openapi.generated.yaml, " +
				"from which the tagString based endpoints will be read and then scripted, " +
				"a wrong readPath can lead to error in the scripting...!")
			return ""
		} else {
			val += "/openapi.generated.yaml"
		}

	case "confPath":
		if val == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter a valid directory path in which you wish to store the conf files for a specific " +
				"testcase : ")
			val, _ = reader.ReadString('\n')
			val = strings.TrimSpace(val)
		}

		// Open the directory
		dir, err := os.Open(val)
		if err != nil {
			fmt.Println("Error opening directory : ", err)
			fmt.Printf("There was an error opening the directory : `%s` , "+
				"please verify if the specified directory is correct or not...!!!", val)
			return ""
		}
		defer func(dir *os.File) {
			err = dir.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(dir)

	case "submodulePath":
		if val == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter a valid path for the submodule directory where the API call functions would be stored, " +
				"which are used by the test scripts.")
			val, _ = reader.ReadString('\n')
			val = strings.TrimSpace(val)
		}

	case "smokeUpgradeDir":
		if val == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter a valid path for the directory where the subdirectories for the three cloud providers" +
				" reside, since the test would automatically append the file based on the --cloud tag to the" +
				" respective directory.")
			val, _ = reader.ReadString('\n')
			val = strings.TrimSpace(val)
		}

		// Open the directory
		dir, err := os.Open(val)
		if err != nil {
			fmt.Println("Error opening directory : ", err)
			fmt.Printf("There was an error opening the directory : `%s` , "+
				"please verify if the specified directory is correct or not...!!!", val)
			return ""
		}
		defer func(dir *os.File) {
			err = dir.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(dir)
	}

	return val
}
