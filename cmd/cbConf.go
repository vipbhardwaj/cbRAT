package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// cbConf represents the cbConf command
var cbConf = &cobra.Command{
	Use:   "cbConf",
	Short: "generates a conf file for the API endpoints that need to be tested",
	Long: `cbConf:
The command can be used to generate a conf file inside the TAF repo.
It would generate conf file based on the manner similar to how it was being generated manually (subject to change).
It would include adding the 'test_api_path' to the appropriate sanity file, which would run in the sanity pipeline.
And it would create a new conf file for that specific endpoint which would include the 'test_authorization', 
'test_multiple_requests_using_API_keys_with_same_role_which_has_access', 
'test_multiple_requests_using_API_keys_with_diff_role' which run in the QE24 pipeline (as of now, April 2024)
There are multiple REQUIRED flags that are mandatory for cbConf to have the complete information of what to be added, 
and how can the script be syntactically the most accurate.

Usage example:
cbConf --tag "Clusters"`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// Check in the MAP populated by cbRat.
		if pathsMap["confPath"] != "" {
			fmt.Println("No valid path found for the conf file to be generated.")
			fmt.Println("!!!...Please set the confPath first using cbPaths...!!!")
		}

		//// If not found there, then it SHOULD have been supplied in the command
		//// earlier by cbPaths
		//f, err := os.Open("./paths.cb")
		//if err != nil {
		//	fmt.Println(err)
		//}
		//defer func(f *os.File) {
		//	err = f.Close()
		//	if err != nil {
		//		fmt.Println(err)
		//	}
		//}(f)
		//
		//scanner := bufio.NewScanner(f)
		//for scanner.Scan() {
		//	line := scanner.Text()
		//	key := strings.Split(line, ":")[0]
		//	if key != "confPath" {
		//		continue
		//	}
		//	val := strings.Split(line, ":")[1]
		//	pathsMap[key] = val
		//	fmt.Println(key, ":", val)
		//	break
		//}
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cbConf started...")

		tagString, _ := cmd.Flags().GetString("tag")
		if tagString == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter the tag string for the endpoints that have to be automated. In openapi.generated.yaml, " +
				"every endpoint has a tag linked to it, and based on the tag that you provide, " +
				"the endpoints under that tag would be automated. " +
				"A multiword tagString input HAS TO be enclosed in quotes : ")
			tagString, _ = reader.ReadString('\n')
			tagString = strings.TrimSpace(tagString)
		}
		if tagString == "" {
			fmt.Println("Please provide a valid tag for which the endpoints linked to it will be scripted, " +
				"(based on the openapi.generated.yaml), a multiword tagString input HAS TO be enclosed in quotes...!")
			return
		}

		nomenclature, _ := cmd.Flags().GetString("nomenclature")
		if nomenclature == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter the directory name in which the test file(" +
				"s) are present for which the conf has to be generated : ")
			nomenclature, _ = reader.ReadString('\n')
			nomenclature = strings.TrimSpace(nomenclature)
		}
		if nomenclature == "" {
			fmt.Println("Please make sure the directory name you are specifying for the conf is valid...!")
			return
		}

		// Calling the python script generator function
		endpoints := specReader(pathsMap["readPath"], tagString, "")
		dirName = removeSpace(removeSpecialChars(tagString)) + "-v4-APIs.conf"
		if err := generateConfFiles(endpoints, nomenclature); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	},
}

func generateConfFiles(endpoints []Endpoint, nomenclature string) error {
	// Construct the destination file path
	destinationFilePath := filepath.Join(pathsMap["confDir"], dirName)
	fmt.Println(destinationFilePath)
	f, err := os.Create(destinationFilePath)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(f)

	//nomenclature := dirName + "-v4-APIs.conf"
	//dirName = strings.Split(nomenclature, "_")[0]
	//destDir = filepath.Join(pathsMap["testPath"], dirName)
	//err := os.MkdirAll(destDir, 0755)
	//if err != nil {
	//	fmt.Println("Error creating directory:", err)
	//	return nil
	//}
	//fmt.Printf("Directory created successfully at %s\n\n", destDir)

	code := `###################################################################################################
# Test GROUPING:
# Tests taking longer than an hour have been removed from being executed in the sanity pipeline.
# Instead they are kept either in the QE24 pipeline tests (current file) or refrained from being in any pipeline at all.

# Therefore a certain Grouping format has been used in all the Capella v4 REST API conf files.
#     - Group RT : The Rate Limiting tests have this associated with them.

# Some Params can be specified while running tests in the pipeline or locally by editing the file or passing them in the test configuration.
#     - server_version : The server version for capella cluster to be deployed. DEFAULT = 7.6
###################################################################################################
`
	for _, endPoint := range endpoints {
		className = toCamelCase(endPoint.method) + firstToUpper(endPoint.endpointParam)
		className = className[:len(className)-1]
		var i int
		switch endPoint.method {
		case "GET":
			i = 0
		case "LIST":
			i = 1
		case "CREATE":
			i = 2
		case "DELETE":
			i = 3
		case "UPDATE":
			i = 4
		}
		code += `
Capella.RestAPIv4.` + nomenclature + `.` + endPoint.fileName + `.` + className + fmt.Sprintf(`:
    test_authorization,GROUP=P%d
    test_multiple_requests_using_API_keys_with_same_role_which_has_access,GROUP=P%d;RT
    test_multiple_requests_using_API_keys_with_diff_role,GROUP=P%d;RT
`, i, i, i)
	}

	_, err = f.WriteString(code)
	if err != nil {
		return err
	}
	fmt.Printf("Conf file '%s' generated successfully.\n", dirName)
	return nil
}
