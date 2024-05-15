package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var dirName string

func generatePythonModule(
	user string,
	commandUsed string,
	superClass string,
	nomenclature string,
	tagString string,
	endpoints []Endpoint) error {

	// Create a module
	// Create the directory with 0755 permissions (read, write, execute for owner, read and execute for others),
	// inside which the individual python test scripts will sit.
	dirName = strings.Split(nomenclature, "_")[0]
	destDir = filepath.Join(pathsMap["modulePath"], dirName)
	err := os.MkdirAll(destDir, 0755)
	if err != nil {
		fmt.Println("Error creating directory:", err)
		return nil
	}
	fmt.Printf("Directory created successfully at %s\n\n", destDir)

	var getTestFile string
	var getTestSuperClass string

	// Call the function to create a python file script one by one :-
	for _, endPoint := range endpoints {
		endpoint = removeSpecialChars(removeSpace(tagString))
		if endpoint[len(endpoint)-1] == 's' {
			endpoint = endpoint[:len(endpoint)-1]
		}
		resourceParam = strings.ToLower(endpoint) + "_id"
		className = toCamelCase(endPoint.method) + firstToUpper(endPoint.endpointParam)
		className = className[:len(className)-1]
		endpointParam := strings.ToLower(camelToSnake(removeSpecialChars(endPoint.endpointParam))) + "_endpoint"

		if endPoint.method == "GET" {
			superClassImport = "from pytests.Capella.RestAPIv4." + convertString(superClass) + "s import " + superClass
			getTestSuperClass = className
			getTestFile = endPoint.fileName
		} else {
			superClass = getTestSuperClass
			superClassImport = "from pytests.Capella.RestAPIv4." + dirName + "." + getTestFile + " import " + superClass
		}

		combinationsParams = [][]string{}
		correctQueryParams = ""
		testApiPathParamInitialization = ""

		nomenclature = strings.Split(nomenclature, "_")[0] + "_" + endPoint.method
		if err = generatePythonFile(user, commandUsed, superClass, nomenclature, endpointParam, endPoint); err != nil {
			fmt.Printf("Error at %s: \n%s", nomenclature, err)
			os.Exit(1)
		}
	}
	return nil
}

// cbModule represents the cbModule command
var cbModule = &cobra.Command{
	Use:   "cbModule",
	Short: "generates a REST API test automation module for an endpoint",
	Long: `cbModule:
The command can be used to create a python file inside a python package.
Where the specified python file would test one of the specific pre-defined REST API endpoint.
Those pre-defined REST APIs are picked based on the flag parameter that is specified by the user.
There are multiple REQUIRED flags that are mandatory for cbModule to have the complete information of what to be tested, and how can the script be syntactically the most accurate.
Still it is advised to manually check the script once in case of anything that cbModule might miss as of now, since it is in the BETA stages.

Usage example:
cbModule --module ClustersXYZ --superclass GetProject --filePath "../foo/bar" --user "Matt Cain" --tag "Clusters"`,
	PreRun: func(cmd *cobra.Command, args []string) {
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
		for key, value := range pathsMap {
			fmt.Println(key, ":", value)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cbModule started...")

		user, _ := cmd.Flags().GetString("user")
		if user == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter the author name for the current automation tool user. eg : Vipul Bhardwaj. " +
				"Person will be held responsible in case of test failures once the script is generated and thoroughly" +
				" checked and tests are ran, verified, and the script code is pushed to GitHub : ")
			user, _ = reader.ReadString('\n')
			user = strings.TrimSpace(user)
		}
		if user == "" {
			fmt.Println("Please provide a valid user who is the testing the current endpoint...!")
			return
		}

		nomenclature, _ := cmd.Flags().GetString("nomenclature")
		if nomenclature == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter nomenclature string for the current endpoint, " +
				"which will be used to name the Resources created in Capella while testing, " +
				"it goes as `ApiToBeTested_RestEndpointTypeThatItUses` : ")
			nomenclature, _ = reader.ReadString('\n')
			nomenclature = strings.TrimSpace(nomenclature)
		}
		if nomenclature == "" {
			fmt.Println("Please provide a nomenclature for the test endpoint...!")
			return
		}

		superClass, _ := cmd.Flags().GetString("superclass")
		if superClass == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter the superclass for the current endpoint : ")
			superClassImport, _ = reader.ReadString('\n')
			superClassImport = strings.TrimSpace(superClassImport)
		}
		if superClass == "" {
			fmt.Println("Provide a valid ClassName...!")
			return
		}

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

		commandUsed := cmd.CommandPath()

		// Calling the python script generator function
		endpoints := specReader(pathsMap["readPath"], tagString, "")
		if err := generatePythonModule(user, commandUsed, superClass, nomenclature, tagString, endpoints); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	},
}
