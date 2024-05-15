package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

func getParams(operation *openapi3.Operation, endpoint *Endpoint) {

	// Request information
	if requestBody := operation.RequestBody; requestBody != nil {
		fmt.Println("Request Body - ")
		for name, example := range operation.RequestBody.Value.Content["application/json"].Examples {
			if example.Value.Value == nil {
				continue
			}
			fmt.Println("\tName :", name)
			jsonString, _ := json.Marshal(example.Value.Value)
			jsonParsedString := prettifyJSON(string(jsonString))
			//fmt.Println("\tExample :", jsonParsedString)
			endpoint.expectedRes = jsonParsedString
		}
		fmt.Println("Payload Params - ")
		fmt.Println(requestBody.Value.Content["application/json"].Schema.Value.Properties)
		for _, param := range requestBody.Value.Content["application/json"].Schema.Value.Required {
			endpoint.requiredParameters = append(endpoint.requiredParameters, param)
		}
	}
	if parameters := operation.Parameters; parameters != nil {
		fmt.Println("Query parameters - ")
		for _, param := range parameters {
			fmt.Println("\t", param.Value.Name)
		}
		if endpoint.method == "GET" {
			endpoint.method = "LIST"
		}
	}

	// Response information
	for statusCode, response := range operation.Responses.Map() {
		if statusCode[0] != '2' {
			continue
		}
		fmt.Println("Response - ")
		fmt.Printf("\tStatus Code : %s\n", statusCode)
		endpoint.statusCode = statusCode
		if response.Value.Content["application/json"] == nil {
			continue
		}
		fmt.Printf("\tResponse Params :\n")
		for _, param := range response.Value.Content["application/json"].Schema.Value.Required {
			fmt.Printf("\t\t%s\n", param)
			if strings.Contains(strings.ToLower(param), "id") {
				if endpoint.expectedIdParam != "" {
					continue
				}
				endpoint.expectedIdParam = param
			}
		}
		fmt.Println("Expected Results - ")
		if len(response.Value.Content["application/json"].Examples) > 0 {
			for _, example := range response.Value.Content["application/json"].Examples {
				if example.Value.Value == nil {
					continue
				}
				jsonParsed, _ := json.Marshal(convert(example.Value.Value))
				jsonParsedString := prettifyJSON(string(jsonParsed))
				//fmt.Printf("\tName : %s\n\tRes : %s\n", name, jsonParsedString)
				//if strings.Contains(strings.ToLower(name), "id") {
				//	endpoint.expectedIdParam = name
				//}
				endpoint.expectedRes = jsonParsedString
			}
		} else if response.Value.Content["application/json"].Example != "" {
			example := response.Value.Content["application/json"].Example
			if example == nil {
				continue
			}
			jsonParsed, _ := json.Marshal(convert(example))
			jsonParsedString := prettifyJSON(string(jsonParsed))
			fmt.Printf("\tRes : %s\n", jsonParsedString)
			endpoint.expectedRes = jsonParsedString
		}
	}
	fmt.Println("-------------------------------------------------")
}
