package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// cbSubmodule represents the cbSubmodule command
var cbSubmodule = &cobra.Command{
	Use:   "cbSubmodule",
	Short: "generates a REST API call functions for an endpoint",
	Long: `cbSubmodule:
The command can be used to append code to a python file inside a repo which is being used as a submodule.
Where the added code would encompass all the possible API calls for one of the specific pre-defined REST API endpoint.
Those pre-defined REST APIs are picked based on the tag parameter that is specified by the user.
There are multiple REQUIRED flags that are mandatory for cbSubmodule to have the complete information of what to be added, and how can the script be syntactically the most accurate.
Still it is advised to manually check the script once in case of anything that cbSubmodule might miss as of now, since it is in the BETA stages.

Usage example:
cbSubmodule --tag "Clusters"`,
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
		//for key, value := range pathsMap {
		//	fmt.Println(key, ":", value)
		//}
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cbSubmodule started...")

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

		//commandUsed := cmd.CommandPath()

		// Calling the python script generator function
		endpoints := specReader(pathsMap["readPath"], tagString, "")
		if err := appendPythonSubModule(endpoints); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	},
}

func appendPythonSubModule(endpoints []Endpoint) error {
	var endpointUrlMap = make(map[string]string)
	var code string
	//var endpointCode string
	for _, endPoint := range endpoints {
		key := strings.Split(endPoint.url, "/")[len(strings.Split(endPoint.url, "/"))-1]
		url := "self." + key + "_endpoint"
		endpointUrlMap[key] = "self." + key + "_endpoint = \"" + endPoint.url + "\""
		fmt.Println(endPoint.url)

		var args string
		for _, arg := range endPoint.payloadParameters {
			args += "\n            " + arg + ","
		}
		args += "\n            headers=None,\n            **kwargs"

		var paramsAssignment = "        params = {"
		for _, param := range endPoint.requiredParameters {
			paramsAssignment += "\n            \"" + param + "\": " + param + ","
		}
		paramsAssignment += "\n        }"

		var params = "                \n"
		var paramsString = "\""
		var kwargs string
		var requestString string
		var endpointCallString string
		for i, param := range endPoint.parameters {
			if i > 0 {
				endpointCallString += ", "
			}
			endpointCallString += param
		}
		if endPoint.method == "POST" {
			params += "name"
			endpointCallString += "name, "
			paramsString += "Creating a " + endPoint.parameters[len(endPoint.parameters)-1][:len(endPoint.
				parameters[len(endPoint.parameters)-1])-2]
			kwargs = `for k, v in kwargs.items():
            params[k] = v`
			requestString += "resp = self.capella_api_" + strings.ToLower(endPoint.method) + "(" + url + ".format(" +
				endpointCallString
		} else if endPoint.method == "PUT" {
			paramsString += "Updating the " + endPoint.parameters[len(endPoint.parameters)-1][:len(endPoint.
				parameters[len(endPoint.parameters)-1])-2]
			kwargs = `for k, v in kwargs.items():
            params[k] = v`
			requestString += "resp = self.capella_api_" + strings.ToLower(endPoint.method) + "(\"{}/{}\".format(" +
				url + ".format(" + endpointCallString
		} else if endPoint.method == "GET" {
			paramsString += "Fetching the " + endPoint.parameters[len(endPoint.parameters)-1][:len(endPoint.
				parameters[len(endPoint.parameters)-1])-2]
			kwargs = `if kwargs:
            params = kwargs
        else:
            params = None`
			requestString += "resp = self.capella_api_" + strings.ToLower(endPoint.method) + "(\"{}/{}\".format(" +
				url + ".format(" + endpointCallString
		} else if endPoint.method == "DELETE" {
			paramsString += "Deleting the " + endPoint.parameters[len(endPoint.parameters)-1][:len(endPoint.
				parameters[len(endPoint.parameters)-1])-2]
			kwargs = `if kwargs:
            params = kwargs
        else:
            params = None`
			requestString += "resp = self.capella_api_del(\"{}/{}\".format(" + url + ".format(" + endpointCallString
		} else if endPoint.method == "LIST" {
			paramsString += "Listing all the " + key + "s" + endpointCallString
			kwargs = `for k, v in kwargs.items():
            params[k] = v`
			requestString += "resp = self.capella_api_" + strings.ToLower(endPoint.method) + "(" + url + ".format(" +
				endpointCallString
		}
		requestString += "), params, headers)\n        return resp"

		for i := len(endPoint.parameters) - 1; i >= 0; i-- {
			if strings.TrimSpace(endPoint.parameters[i]) != "" {
				params += ", "
			}
			params += endPoint.parameters[i]
			paramsString += " in " + endPoint.parameters[i][:len(endPoint.parameters[i])-2] + " {}"
		}
		code += fmt.Sprintf(`
        """
        %s
        """    def %s(%s):`, endPoint.description, endPoint.funcName, args)
		code += fmt.Sprintf(`
        self.cluster_ops_API_log.info(%s.format(%s))
        %s
        %s

        %s

        %s`, paramsString, params, paramsAssignment,
			//optionalParams,
			kwargs, requestString)
	}
	return nil
}
