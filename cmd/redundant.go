package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"unicode"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v2"
)

func attachExpectedResToGet(endpoints []Endpoint) {
	reverseSlice(endpoints)
	for i, endPoint := range endpoints {
		if endPoint.expectedRes != "" && endPoint.method != "GET" {
			fmt.Println("attaching from", endPoint.funcName)
			attachToGet(endpoints, endPoint)
			endpoints[i].expectedRes = ""
		}
	}
	reverseSlice(endpoints)
}

func attachToGet(endpoints []Endpoint, endpoint Endpoint) {

	// This keeps a mapping of URLs to deletion codes mapping.
	deletionCodes := make(map[string]string)

	for i, endPoint := range endpoints {
		if endPoint.method == "DELETE" {
			deletionCodes[endPoint.url] = endPoint.statusCode
			continue
		}
		if endPoint.method != "GET" {
			continue
		}
		var urlGet = endPoint.url
		var payloadStringForCreation string
		lastParam := strings.Split(endPoint.url, "/")[len(strings.Split(endPoint.url, "/"))-1]
		if lastParam[0] == '{' {
			urlGet = endPoint.url[:len(lastParam)+1]
		}
		if urlGet == endpoint.url {
			for url, deleteCode := range deletionCodes {
				if url == urlGet {
					endpoints[i].deleteResponseCode = deleteCode
				}
			}
			for j, param := range endpoint.requiredParameters {
				if j > 0 {
					payloadStringForCreation += ","
				}
				payloadStringForCreation += "\n            self.expected_res[\"" + param + "\"]"
				payloadStringForCreationIndented += ",\n                    self.expected_res[\"" + param + "\"]"
			}
			endpoints[i].expectedRes = endpoint.expectedRes
			endpoints[i].createPayload = payloadStringForCreation
			endpoints[i].createResponseCode = endpoint.statusCode
			endpoints[i].responseIdParam = endpoint.expectedIdParam
			break
		}
	}
	//for i, endPoint := range endpoints {
	//	if endPoint.method == "POST" {
	//
	//	}
	//}
}

func readFile(filePath string, method string, opId string) []byte {
	//if method != "PUT" && method != "POST" {
	//	return []byte{}, []string{}
	//}

	// Open the OpenAPI specification file
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("failed to open file: %v", err)
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			return
		}
	}(file)

	scanner := bufio.NewScanner(file)
	var lines [][]byte
	var flagPayloadParams = false
	var flagReachedOp = false

	// Read lines from the file
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "operationId: "+opId {
			flagReachedOp = true
		}
		if flagReachedOp && strings.TrimSpace(scanner.Text()) == "value:" {
			if method == "PUT" || method == "POST" {
				flagPayloadParams = true
			}
			continue
		}
		if flagPayloadParams {
			if strings.TrimSpace(scanner.Text()) == "responses:" {
				break
			}
			payloadParam := scanner.Text()
			payloadParam = strings.TrimSpace(strings.Split(payloadParam, ":")[0])
			payloadParams += "\n                self.expected_res[\"" + payloadParam + "\"],"
			payloadParamsIndented += "\n                    self.expected_res[\"" + payloadParam + "\"],"
			//line := scanner.Bytes()
			//lines = append(lines, line)
		}
	}
	// Check for errors during scanning
	if err = scanner.Err(); err != nil {
		log.Fatalf("error reading file: %v", err)
	}
	joined := bytes.Join(lines, []byte("\n"))
	return joined
}

func getExpectedRes(yamlData []byte) string {
	// Convert YAML to JSON
	jsonData, err := yamlToJson(yamlData)
	if err != nil {
		log.Fatalf("failed to convert YAML to JSON: %v", err)
	}

	return string(jsonData)
}

func getSuccessCodeForAPI(op *openapi3.Operation) string {
	var successCode string
	for statusCode, response := range op.Responses.Map() {
		if statusCode[0] != '2' {
			continue
		}
		successCode = statusCode
		if response.Ref != "" {
			refResponse := &ResponseDescription{}
			err := yaml.Unmarshal([]byte(response.Ref), refResponse)
			if err != nil {
				log.Fatalf("failed to parse response reference: %v", err)
			}
			fmt.Println(refResponse.Description)
		}
	}
	return successCode
}

func yamlToJson(yamlData []byte) ([]byte, error) {
	var data interface{}

	// Unmarshal YAML data into a generic interface
	err := yaml.Unmarshal(yamlData, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	data = convert(data)

	// Marshal the generic interface into JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %v", err)
	}
	return jsonData, nil
}

func convert(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = convert(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convert(v)
		}
	}
	return i
}

func prettifyJSON(jsonString string) string {
	// Unmarshal the JSON string into an interface{} value
	var data interface{}
	err := json.Unmarshal([]byte(jsonString), &data)
	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}

	// Marshal the data back to JSON with indentation for better readability
	formattedJSON, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}

	// Create a buffer to hold the indented JSON
	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, formattedJSON, "", "    ")
	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}

	var indentedJSON string
	for i, line := range strings.Split(prettyJSON.String(), "\n") {
		if i == 0 {
			indentedJSON += line
			continue
		}
		indentedJSON += "\n        " + line
	}
	return indentedJSON
}

func removeSpecialChars(input string) string {
	var result string

	for _, char := range input {
		if unicode.IsLetter(char) {
			result += string(char)
		}
	}

	return result
}

func reverseSlice(slice []Endpoint) {
	for i := 0; i < len(slice)/2; i++ {
		j := len(slice) - i - 1
		slice[i], slice[j] = slice[j], slice[i]
	}
}

func convertString(input string) string {
	// Convert the string to lowercase
	converted := strings.ToLower(input)

	converted = camelToSnake(input)
	prefix := getPrefix(converted)
	return prefix + "s." + strings.ToLower(converted)
}

func camelToSnake(s string) string {
	var builder strings.Builder

	for i, r := range s {
		if i > 0 && ('A' <= r && r <= 'Z') {
			builder.WriteRune('_')
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func getPrefix(input string) string {
	parts := strings.Split(input, "_")
	parts = parts[1:]
	var prefix string
	for _, part := range parts {
		if len(part) > 0 {
			prefix += strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return prefix
}

func removeSpace(str string) string {
	var newStr string
	for _, char := range str {
		if char == ' ' {
			continue
		}
		newStr += string(char)
	}
	return newStr
}

func toCamelCase(s string) string {
	// Split the string into words
	words := strings.Fields(s)

	// Convert the first character of each word to uppercase
	for i, word := range words {
		word = strings.ToLower(word)
		words[i] = firstToUpper(word)
	}

	// Join the words to form the camel case string
	camelCaseStr := strings.Join(words, "")

	return camelCaseStr
}

func firstToUpper(s string) string {
	// Convert the first character of the string to uppercase
	if s == "" {
		return ""
	}
	r := []rune(s)
	return string(unicode.ToUpper(r[0])) + string(r[1:])
}

func removeTrailingSpaces(lines []string) string {
	// Remove trailing spaces from each line
	for i, line := range lines {
		lines[i] = strings.TrimRightFunc(line, unicode.IsSpace)
	}

	// Join the lines back together
	cleanedStr := strings.Join(lines, "\n")
	return cleanedStr
}
