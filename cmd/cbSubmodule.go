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
	for _, endPoint := range endpoints {
		key := strings.Split(endPoint.url, "/")[len(strings.Split(endPoint.url, "/"))-1]
		url := "self." + key + "_endpoint"
		endpointUrlMap[key] = "self." + key + "_endpoint = \"" + endPoint.url + "\""
		fmt.Println(endPoint.url)

		args := "\n            self"
		var argsDesc string
		var paramsAssignment = "params = {"
		var endpointCallString string
		for i, arg := range append(
			append(endPoint.parameters,
				endPoint.requiredParameters...), []string{"headers", "**kwargs"}...) {
			args += ",\n        " + arg
			argsDesc += "        " + arg + ": "
			switch arg {
			case "organizationId":
				argsDesc += "The tenant ID for the path. (UUID)\n"
			case "projectId":
				argsDesc += "ID of the project inside the tenant. (UUID)\n"
			case "clusterId":
				argsDesc += "ID of the cluster which has the app service inside it. (UUID)\n"
			case "appServiceId":
				argsDesc += "ID of the app service linked to the cluster. (UUID)\n"
			case "bucketId":
				argsDesc += "ID of the bucket inside the cluster. (string)\n"
			case "payload":
				argsDesc += "Payload string coming into `params`. (string)\n"
			case "headers":
				args += "=None"
				argsDesc += "Headers to be sent with the API call. (dict)\n"
			case "**kwargs":
				argsDesc += "Do not use this under normal circumstances. This is only to test negative scenarios. " +
					"(dict)\n"
			default:
				argsDesc += "\n"
			}
			if i < len(endPoint.parameters) {
				if i > 0 {
					endpointCallString += ", "
				}
				endpointCallString += arg
			}
		}
		//for _, arg := range endPoint.payloadParameters {
		//	args += "\n            " + arg + ","
		//}
		for _, param := range endPoint.requiredParameters {
			paramsAssignment += "\n            \"" + param + "\": " + param + ","
			//args += "\n        " + param + ","
			//argsDesc += "        " + param + ": \n"
			//endpointCallString += ", " + param
		}
		//args += "\n        headers=None,\n        **kwargs"
		//argsDesc += "        headers: Headers to be sent with the API call. (dict)\n        **kwargs: \n"
		paramsAssignment += "\n        }"

		var params = "                \n"
		var paramsString = "\""
		var kwargs string
		if len(endPoint.requiredParameters) > 0 {
			kwargs = `for k, v in kwargs.items():
        params[k] = v`
		} else {
			paramsAssignment = ""
			kwargs = `if kwargs:
        params = kwargs
    else:
        params = None`
		}

		var requestString string
		if endPoint.method == "POST" {
			paramsString += "Creating a " + endPoint.parameters[len(endPoint.parameters)-1][:len(endPoint.
				parameters[len(endPoint.parameters)-1])-2]
			requestString += "resp = self.api_" + strings.ToLower(endPoint.method) + "(" + url + ".format(" +
				endpointCallString
		} else if endPoint.method == "PUT" {
			paramsString += "Updating the " + endPoint.parameters[len(endPoint.parameters)-1][:len(endPoint.
				parameters[len(endPoint.parameters)-1])-2]
			requestString += "resp = self.api_" + strings.ToLower(endPoint.method) + "(\"{}/{}\".format(" +
				url + ".format(" + endpointCallString
		} else if endPoint.method == "GET" {
			paramsString += "Fetching the " + endPoint.parameters[len(endPoint.parameters)-1][:len(endPoint.
				parameters[len(endPoint.parameters)-1])-2]
			requestString += "resp = self.api_" + strings.ToLower(endPoint.method) + "(\"{}/{}\".format(" +
				url + ".format(" + endpointCallString
		} else if endPoint.method == "DELETE" {
			paramsString += "Deleting the " + endPoint.parameters[len(endPoint.parameters)-1][:len(endPoint.
				parameters[len(endPoint.parameters)-1])-2]
			requestString += "resp = self.api_del(\"{}/{}\".format(" + url + ".format(" + endpointCallString
		} else if endPoint.method == "LIST" {
			paramsString += "Listing all the " + key + "s" + endpointCallString
			requestString += "resp = self.api_get(" + url + ".format(" +
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
    def %s(%s):
    """
%s

    Args:
%s
    Returns:
        Success : Status Code and response (JSON).
        Error : message, hint, code, HttpStatusCode
    """`, endPoint.funcName, args, endPoint.description, argsDesc)
		code += fmt.Sprintf(`
    self.cluster_ops_API_log.info(%s".format(%s))
    %s
    %s

    %s`, paramsString, params, paramsAssignment,
			//optionalParams,
			kwargs, requestString)
	}
	fmt.Printf("Please copy and paste the following:\n```\n" + code + "\n```")
	return nil
}
