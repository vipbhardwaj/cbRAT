package cmd

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

type Endpoint struct {
	url                string
	api                string
	description        string
	method             string
	parameters         []string
	statusCode         string
	endpointParam      string
	expectedRes        string
	fileName           string
	funcName           string
	createPayload      string
	payloadParameters  []string
	requiredParameters []string
	createResponseCode string
	deleteResponseCode string
	expectedIdParam    string
	responseIdParam    string
	authorizedRoles    []string
}
type ResponseDescription struct {
	Description string `yaml:"description"`
	Ref         string `yaml:"$ref"`
}

var pythonRoleToGoRoleMapper = map[string]string{
	"Organization Member":         "organizationMember",
	"Project Creator":             "projectCreator",
	"Organization Owner":          "organizationOwner",
	"Project Owner":               "projectOwner",
	"Project Manager":             "projectManager",
	"Project Viewer":              "projectViewer",
	"Database Data Reader/Writer": "projectDataReaderWriter",
	"Database Data Reader":        "projectDataReader",
}
var payloadParams string
var payloadParamsIndented string
var payloadStringForCreationIndented string

func specReader(filePath string, tagString string, operationId string) []Endpoint {
	// Open the OpenAPI specification file
	spec, err := openapi3.NewLoader().LoadFromFile(filePath)
	if err != nil {
		log.Fatalf("error loading OpenAPI specification: %s", err)
	}

	if operationId != "" {
		return getEndpoint(spec, operationId)
	} else {
		return getEndpoints(spec, tagString)
	}
}

func getEndpoint(spec *openapi3.T, operationId string) []Endpoint {
	var endPoints []Endpoint
	for path, pathItem := range spec.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			if operationId != operation.OperationID {
				continue
			}
			endPoints = append(endPoints, appendEndPoint(path, method, operation))
		}
	}
	if len(endPoints) == 0 {
		fmt.Println("ERROR : No operation found by the ID :", operationId)
	} else if len(endPoints) > 1 {
		fmt.Println("WARNING : Multiple endpoints found by the ID :", operationId)
		return []Endpoint{}
	}
	return endPoints
}

func getEndpoints(spec *openapi3.T, tagString string) []Endpoint {
	var endPoints []Endpoint
	// Access path parameters for each API endpoint
	for path, pathItem := range spec.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			if operation.Tags[0] != tagString {
				continue
			}
			endPoints = append(endPoints, appendEndPoint(path, method, operation))
		}
	}

	// If user provided a wrong tagString val for tag flag.
	if len(endPoints) < 1 {
		fmt.Printf("No endpoints found in: \"%s\" linked to the tag: \"%s\"\n", pathsMap["readPath"], tagString)
	} else if len(endPoints) > 1 {
		// Custom sorting function
		sort.Slice(endPoints, func(i, j int) bool {
			if endPoints[i].method == "GET" {
				return true
			} else if endPoints[j].method == "GET" {
				return false
			}
			return endPoints[i].method < endPoints[j].method
		})
		attachExpectedResToGet(endPoints)
	}
	return endPoints
}

func appendEndPoint(path string, method string, operation *openapi3.Operation) Endpoint {
	var endPoint Endpoint

	endPoint.description = descTrimmer(operation.Description)

	endPoint.api = "capellaAPI.cluster_ops_apis"
	for _, e := range strings.Split(path, "/") {
		if e == "analyticsClusters" {
			endPoint.api = "columnarAPI"
			break
		}
	}

	fmt.Println("Operation Name :", operation.OperationID)
	endPoint.fileName = strings.ToLower(camelToSnake(removeSpecialChars(operation.OperationID)))
	endPoint.funcName = endPoint.fileName
	endPoint.endpointParam = getEndpointParam(path)
	fmt.Println("Endpoint Param :::::: ", endPoint.endpointParam)

	urlPath := getUrlPath(path)
	endPoint.method = method
	endPoint.url = urlPath

	parameters := getApiParameters(path)
	fmt.Println(endPoint.method, " - ", parameters)
	//fmt.Println(operation.Responses)
	endPoint.parameters = parameters

	// Get Authorized Roles
	authorizedRoles := getAuthorizedRolesForAPI(operation, endPoint)
	endPoint.authorizedRoles = authorizedRoles

	/////////// DEPRECATED /////////

	//// Get Success Codes
	//successCode := getSuccessCodeForAPI(operation)
	//endPoint.statusCode = successCode

	//// Read YAML - it returns the
	//// expectedRes (For GET class), payload params (for POST & PUT files),
	//// in that order
	//yamlStr := readFile(filePath, method, operation.OperationID)
	//if len(yamlStr) != 0 {
	//	endPoint.expectedRes = prettifyJSON(getExpectedRes(yamlStr))
	//}

	////////////////////////////////

	getParams(operation, &endPoint)

	//endPoint.funcName = strings.ToLower(camelToSnake(removeSpecialChars(operation.OperationID)))
	if endPoint.method == "GET" && strings.Split(endPoint.funcName, "_")[0] != "list" {
		endPoint.funcName = "fetch_" + strings.Join(strings.Split(endPoint.funcName, "_")[1:], "_") + "_info"
	} else if endPoint.method == "PUT" && strings.Split(endPoint.funcName, "_")[0] != "update" {
		endPoint.funcName = "update_" + strings.ToLower(camelToSnake(removeSpecialChars(endPoint.endpointParam)))
		endPoint.fileName = "update_" + strings.Join(strings.Split(endPoint.fileName, "_")[1:], "_") + "s"
	} else if endPoint.method == "POST" {
		if strings.Split(endPoint.fileName, "_")[0] == "post" {
			endPoint.funcName = "create_" + strings.ToLower(camelToSnake(removeSpecialChars(endPoint.endpointParam)))
			endPoint.fileName = "create_" + strings.Join(strings.Split(endPoint.fileName, "_")[1:], "_") + "s"
		} else {
			if strings.Split(endPoint.funcName, "_")[len(strings.Split(endPoint.funcName, "_"))-1] == "on" {
				endPoint.funcName = "turn_" + endPoint.funcName
			}
		}
	} else if endPoint.method == "DELETE" && endPoint.fileName[len(endPoint.fileName)-1] != 's' {
		if strings.Split(endPoint.funcName, "_")[len(strings.Split(endPoint.funcName, "_"))-1] == "off" {
			endPoint.funcName = "turn_" + endPoint.funcName
		} else {
			endPoint.fileName += "s"
		}
	}
	if endPoint.method != "LIST" && endPoint.funcName[len(endPoint.funcName)-1] == 's' {
		endPoint.funcName = endPoint.funcName[:len(endPoint.funcName)-1]
	}
	if endPoint.fileName[len(endPoint.fileName)-1] != 's' {
		endPoint.fileName += "s"
	}

	return endPoint
}

func descTrimmer(fullDesc string) string {
	var infoString strings.Builder
	for _, line := range strings.Split(fullDesc, "\n") {
		if line == "In order to access this endpoint, "+
			"the provided API key must have at least one of the roles referenced below:" {
			break
		}
		infoString.WriteString(line)
	}
	return infoString.String()
}

func getUrlPath(path string) string {
	flag := true
	var urlPath string
	for _, c := range path {
		if c == '}' {
			flag = true
		}
		if flag {
			urlPath += string(c)
		} else {
			continue
		}
		if c == '{' {
			flag = false
		}
	}
	// Return True or False for the Method type being API call being LIST for a GET method.
	if urlPath[len(urlPath)-1] == '}' {
		urlPath = urlPath[:len(urlPath)-3]
		return urlPath
	}
	return urlPath
}

func getApiParameters(path string) []string {
	var parameters []string
	for _, param := range strings.Split(path, "/") {
		if len(param) == 0 || param[0] != '{' {
			continue
		}
		parameters = append(parameters, param[1:len(param)-1])
	}
	return parameters
}

func getAuthorizedRolesForAPI(op *openapi3.Operation, endpoint Endpoint) []string {
	var authorizedRoles []string
	desc := op.Description
	lines := strings.Split(desc, "\n")
	attachDescription(endpoint, lines)
	for _, line := range lines {
		if len(line) == 0 || strings.TrimSpace(line)[0] != '-' {
			continue
		}
		role := strings.TrimSpace(line)[2:]
		authorizedRoles = append(authorizedRoles, pythonRoleToGoRoleMapper[role])
	}
	return authorizedRoles
}

func attachDescription(endpoint Endpoint, lines []string) {
	for _, line := range lines {
		if len(line) >= 6 && string(line[len(line)-7:]) == "html)." {
			return
		}
		endpoint.description += "            " + line + "\n"
	}
}

func getEndpointParam(path string) string {
	// Split the path by slashes
	params := strings.Split(path, "/")

	// Get the last slash param, and verify if that is a path string or param.
	if params[len(params)-1][0] == '{' {
		return params[len(params)-2]
	} else {
		return params[len(params)-1]
	}
}
