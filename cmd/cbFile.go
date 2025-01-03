package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func fetchApiPathTestcasesFromSuperClass(parameters []string) string {
	var testcases string
	// Read the file line by line
	for _, param := range parameters {
		paramId := param[:len(param)-2] + "_id"
		if paramId == "organization_id" {
			paramId = "organisation_id"
		}
		testcases += fmt.Sprintf(`            }, {
                "description": "Call API with non-hex %s",
                "invalid_%s": self.replace_last_character(
                    self.%s, non_hex=True),
                "expected_status_code": 400,
                "expected_error": {
                    "code": 1000,
                    "hint": "Check if you have provided a valid URL and all "
                            "the required params are present in the request "
                            "body.",
                    "httpStatusCode": 400,
                    "message": "The server cannot or will not process the "
                               "request due to something that is perceived to "
                               "be a client error."
                }
`, param, param, paramId)
	}
	testcases += "            }\n        ]"
	return testcases
}

var superClassImport string
var destDir string

// var apiType string
var endpoint string
var resourceParam string
var className string
var setUp string
var tearDown string
var apiPathFuncCallParams string
var testApiPathParamInitialization string
var authorizationFuncCallParams []string
var correctQueryParams string
var combinationsParams [][]string

func generateSetupAndTeardown(nomenclature string, superClass string, endPoint Endpoint, params string) string {

	var SetupAndTearDown string
	setUp = fmt.Sprintf(`

    def setUp(self, nomenclature="%s"):
        %s.setUp(self, nomenclature)
`, nomenclature, superClass)

	tearDown = `
    def tearDown(self):`

	temp := strings.ToLower(camelToSnake(endpoint))
	if endPoint.expectedRes != "" {
		setUp += fmt.Sprintf(`
        self.expected_res = %s

        self.log.info("Creating %s")
        res = self.%s.create_%s(
            %s%s
        )
        if res.status_code != %s:
            self.log.error("Result: {}".format(res.content))
            self.tearDown()
            self.fail("Error while creating %s.")
        self.log.info("%s created successfully.")
`, endPoint.expectedRes, endpoint, endPoint.api, temp, params, endPoint.createPayload, endPoint.createResponseCode,
			endpoint, endpoint)

		if endPoint.responseIdParam != "" {
			var endResourceParam = endPoint.parameters[len(endPoint.parameters)-1]
			endResourceParam = endResourceParam[:len(endResourceParam)-2] + "_id"
			setUp += fmt.Sprintf(`
        self.%s = res.json()["%s"]
        self.expected_res["%s"] = self.%s
        ##################################################
        # Set Parameters for the expected res to be validated in GET
        ##################################################
`, endResourceParam, endPoint.responseIdParam, endPoint.expectedIdParam, endResourceParam)
		}

		var deletionParams string
		for _, param := range endPoint.parameters[1:] {
			deletionParams += ",\n            self." + param[:len(param)-2] + "_id"
		}
		tearDown += fmt.Sprintf(`
        self.update_auth_with_api_token(self.org_owner_key["token"])

        # Delete the %s.
        self.log.info("Deleting the %s")
        res = self.%s.delete_%s(
            self.organisation_id%s)
        if res.status_code != %s:
            self.log.error("Result: {}".format(res.content))
            self.tearDown()
            self.fail("%s deletion failed")
        self.log.info("Successfully deleted the %s.")
`, endpoint, endpoint, endPoint.api, temp, deletionParams, endPoint.deleteResponseCode, endpoint, endpoint)
	} else if endPoint.method == "LIST" {
		setUp += `
        self.expected_res = {
            "cursor": {
                "hrefs": {
                    "first": None,
                    "last": None,
                    "next": None,
                    "previous": None
                },
                "pages": {
                    "last": None,
                    "next": None,
                    "page": None,
                    "perPage": None,
                    "previous": None,
                    "totalItems": None
                }
            },
            "data": [
                self.expected_res
            ]
        }
`
	}

	tearDown += fmt.Sprintf(`
        super(%s, self).tearDown()`, className)

	SetupAndTearDown = setUp + tearDown
	return SetupAndTearDown
}

func getExtraValidationParams(endPoint Endpoint) string {
	var deleteTheCreatedParams string
	var createToTestParams string

	for i, param := range authorizationFuncCallParams[1:] {
		if i > 0 && i%2 == 0 {
			createToTestParams += "\n                    "
			deleteTheCreatedParams += "\n                                    "
		}
		if i < len(authorizationFuncCallParams[1:])-1 {
			createToTestParams += "\n                    " + param
		}
		deleteTheCreatedParams += param + ", "
	}
	if len(authorizationFuncCallParams[1:])%2 == 0 {
		deleteTheCreatedParams += "\n                                    "
	}

	var extraParams string
	if endPoint.responseIdParam != "" {
		extraParams = fmt.Sprintf(`, True,
                                   self.expected_res, self.%s`, endPoint.parameters[len(endPoint.parameters)-1][:len(
			endPoint.parameters[len(endPoint.parameters)-1])-2]+"_id")
	} else if endPoint.method == "POST" {
		extraParams = fmt.Sprintf(`):
                self.log.debug("Creation Successful")
                self.flush_%ss(%s[result.json()["id"]]`, resourceParam[:len(resourceParam)-3], deleteTheCreatedParams)
	} else if endPoint.method == "DELETE" {
		extraParams = fmt.Sprintf(`):
                self.log.debug("Deletion Successful")
                self.%s_id = self.create_%s_to_be_tested(%s%s`, resourceParam[:len(resourceParam)-3],
			resourceParam[:len(resourceParam)-3], createToTestParams, payloadStringForCreationIndented)
	}
	extraParams += `)`
	return extraParams
}

func testPathParams(parameters []string) string {
	var apiCallParamsInstantiation []string
	var testcases string
	var paramTrimmed string

	for i, param := range parameters {
		if i == 0 {
			apiCallParamsInstantiation = append(apiCallParamsInstantiation, "organization")
			continue
		}
		paramTrimmed = param[:len(param)-2]
		testcase := fmt.Sprintf(`
            elif "invalid_%s" in testcase:
                %s = testcase["invalid_%s"]`, param, paramTrimmed, param)
		testcases += testcase
		apiCallParamsInstantiation = append(apiCallParamsInstantiation, paramTrimmed)
	}
	apiPathFuncCallParams = ""
	authorizationFuncCallParams = []string{}
	for i, apiCallParam := range apiCallParamsInstantiation {
		apiPathFuncCallParams += apiCallParam + ", "
		if apiCallParam == "organization" {
			authorizationFuncCallParams = append(authorizationFuncCallParams, "self.organisation_id")
			testApiPathParamInitialization += "            " + apiCallParam + " = self.organisation_id\n"
		} else {
			authorizationFuncCallParams = append(authorizationFuncCallParams, "self."+apiCallParam+"_id")
			testApiPathParamInitialization += "            " + apiCallParam + " = self." + apiCallParam + "_id\n"
		}
		correctQueryParams += apiCallParam + " ID: {}, "
		if i > 0 && i%2 != 0 && i != len(apiCallParamsInstantiation)-1 {
			//correctQueryParams = correctQueryParams[:len(correctQueryParams)-1]
			correctQueryParams += "\"\n                \""
		}
		testcaseParam := apiCallParam + "ID"
		firstCapitalApiCallParam := strings.ToUpper(string(apiCallParam[0])) + apiCallParam[1:] + "ID"
		combination := fmt.Sprintf(`combination[%d]`, i)
		combinationsParams = append(combinationsParams,
			[]string{firstCapitalApiCallParam, testcaseParam, combination})
	}
	apiPathFuncCallParams = apiPathFuncCallParams[:len(apiPathFuncCallParams)-2]
	correctQueryParams = correctQueryParams[:len(correctQueryParams)-2]
	return testcases
}

func generateElifsForQueryTests() string {
	var elifs string
	if len(combinationsParams) == 2 {
		elifs = `
                else:
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = {
                        "code": 2000,
                        "hint": "Check if the project ID is valid.",
                        "httpStatusCode": 404,
                        "message": "The server cannot find a project by its "
                                   "ID."
                    }`
	} else if len(combinationsParams) == 3 {
		if endpoint == "Alerts" {
			elifs = `
                elif combination[1] != self.project_id:
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = {
                        "code": 2000,
                        "hint": "Check if the project ID is valid.",
                        "httpStatusCode": 404,
                        "message": "The server cannot find a project by its "
                                   "ID."
                    }
                else:
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = {
                        "code": 3000,
                        "hint": "Please ensure that the organization ID is "
                                "correct.",
                        "httpStatusCode": 404,
                        "message": "Not Found."
                    }`
		} else {
			elifs = `
                elif combination[2] != self.cluster_id:
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = {
                        "code": 4025,
                        "hint": "The requested cluster details could not be "
                                "found or fetched. Please ensure that the "
                                "correct cluster ID is provided.",
                        "message": "Unable to fetch the cluster details.",
                        "httpStatusCode": 404
                    }
                else:
                    testcase["expected_status_code"] = 422
                    testcase["expected_error"] = {
                        "code": 4031,
                        "hint": "Please provide a valid projectId.",
                        "httpStatusCode": 422,
                        "message": "Unable to process the request. The "
                                   "provided projectId {} is not valid for "
                                   "the cluster {}."
                        .format(combination[1], combination[2])
                    }`
		}
	} else if len(combinationsParams) == 4 {
		if endpoint == "AuditLogs" {
			elifs = `
                elif combination[1] != self.project_id:
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = {
                        "code": 2000,
                        "hint": "Check if the project ID is valid.",
                        "httpStatusCode": 404,
                        "message": "The server cannot find a project by its "
                                   "ID."
                    }
                else:
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = {
                        "code": 3000,
                        "hint": "Please ensure that the organization ID is "
                                "correct.",
                        "httpStatusCode": 404,
                        "message": "Not Found."
                    }`
		} else if endpoint == "AuditLogExports" {
			elifs = `
                elif combination[2] != self.cluster_id:
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = {
                        "code": 4025,
                        "hint": "The requested cluster details could not be "
                                "found or fetched. Please ensure that the "
                                "correct cluster ID is provided.",
                        "message": "Unable to fetch the cluster details.",
                        "httpStatusCode": 404
                    }
                elif combination[2] != self.project_id:
                    testcase["expected_status_code"] = 422
                    testcase["expected_error"] = {
                        "code": 4031,
                        "hint": "Please provide a valid projectId.",
                        "httpStatusCode": 422,
                        "message": "Unable to process the request. The "
                                   "provided projectId {} is not valid for "
                                   "the cluster {}."
                        .format(combination[1], combination[2])
                    }
                else:
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = {
                        "code": 404,
                        "hint": "Please review your request and ensure that "
                                "all required parameters are correctly "
                                "provided.",
                        "httpStatusCode": 404,
                        "message": "The requested export ID does not exist."
                    }`
		} else {
			elifs = `
                elif combination[3] != self.bucket_id and not \
                        isinstance(combination[3], type(None)):
                    testcase["expected_status_code"] = 400
                    testcase["expected_error"] = {
                        "code": 400,
                        "hint": "Please review your request and ensure "
                                "that all required parameters are "
                                "correctly provided.",
                        "message": "BucketID is invalid.",
                        "httpStatusCode": 400
                    }
                elif combination[2] != self.cluster_id:
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = {
                        "code": 4025,
                        "hint": "The requested cluster details could not be "
                                "found or fetched. Please ensure that the "
                                "correct cluster ID is provided.",
                        "message": "Unable to fetch the cluster details.",
                        "httpStatusCode": 404
                    }
                elif combination[1] != self.project_id:
                    testcase["expected_status_code"] = 422
                    testcase["expected_error"] = {
                        "code": 4031,
                        "hint": "Please provide a valid projectId.",
                        "httpStatusCode": 422,
                        "message": "Unable to process the request. The "
                                   "provided projectId {} is not valid for "
                                   "the cluster {}."
                        .format(combination[1], combination[2])
                    }
                elif isinstance(combination[3], type(None)):
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = {
                        "code": 6008,
                        "hint": "The requested bucket does not exist. Please "
                                "ensure that the correct bucket ID is "
                                "provided.",
                        "httpStatusCode": 404,
                        "message": "Unable to find the specified bucket."
                    }
                else:
                    testcase["expected_status_code"] = 400
                    testcase["expected_error"] = {
                        "code": 400,
                        "hint": "Please review your request and ensure that "
                                "all required parameters are correctly "
                                "provided.",
                        "message": "BucketID is invalid.",
                        "httpStatusCode": 400
                    }`
		}
	} else if len(combinationsParams) == 5 {
		elifs = `
                elif combination[3] != self.bucket_id and not \
                        isinstance(combination[3], type(None)):
                    testcase["expected_status_code"] = 400
                    testcase["expected_error"] = {
                        "code": 400,
                        "hint": "Please review your request and ensure that "
                                "all required parameters are correctly "
                                "provided.",
                        "message": "BucketID is invalid.",
                        "httpStatusCode": 400
                    }
                elif combination[2] != self.cluster_id:
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = {
                        "code": 4025,
                        "hint": "The requested cluster details could not be "
                                "found or fetched. Please ensure that the "
                                "correct cluster ID is provided.",
                        "message": "Unable to fetch the cluster details.",
                        "httpStatusCode": 404
                    }
                elif combination[1] != self.project_id:
                    testcase["expected_status_code"] = 422
                    testcase["expected_error"] = {
                        "code": 4031,
                        "hint": "Please provide a valid projectId.",
                        "httpStatusCode": 422,
                        "message": "Unable to process the request. The "
                                   "provided projectId {} is not valid for "
                                   "the cluster {}."
                        .format(combination[1], combination[2])
                    }
                elif isinstance(combination[3], type(None)):
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = {
                        "code": 6008,
                        "hint": "The requested bucket does not exist. Please "
                                "ensure that the correct bucket ID is "
                                "provided.",
                        "httpStatusCode": 404,
                        "message": "Unable to find the specified bucket."
                    }
                else:
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = {
                        "code": 11002,
                        "hint": "The requested scope details could not be "
                                "found or fetched. Please ensure that the "
                                "correct scope name is provided.",
                        "httpStatusCode": 404,
                        "message": "Scope Not Found"
                    }`
	} else if len(combinationsParams) == 6 {
		elifs = `
                elif combination[3] != self.bucket_id and not \
                        isinstance(combination[3], type(None)):
                    testcase["expected_status_code"] = 400
                    testcase["expected_error"] = {
                        "code": 400,
                        "hint": "Please review your request and ensure that "
                                "all required parameters are correctly "
                                "provided.",
                        "message": "BucketID is invalid.",
                        "httpStatusCode": 400
                    }
                elif combination[2] != self.cluster_id:
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = {
                        "code": 4025,
                        "hint": "The requested cluster details could not be "
                                "found or fetched. Please ensure that the "
                                "correct cluster ID is provided.",
                        "message": "Unable to fetch the cluster details.",
                        "httpStatusCode": 404
                    }
                elif combination[1] != self.project_id:
                    testcase["expected_status_code"] = 422
                    testcase["expected_error"] = {
                        "code": 4031,
                        "hint": "Please provide a valid projectId.",
                        "httpStatusCode": 422,
                        "message": "Unable to process the request. The "
                                   "provided projectId {} is not valid for "
                                   "the cluster {}."
                        .format(combination[1], combination[2])
                    }
                elif isinstance(combination[3], type(None)):
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = {
                        "code": 6008,
                        "hint": "The requested bucket does not exist. Please "
                                "ensure that the correct bucket ID is "
                                "provided.",
                        "httpStatusCode": 404,
                        "message": "Unable to find the specified bucket."
                    }
                elif combination[4] != self.scope_name:
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = {
                        "code": 11002,
                        "hint": "The requested scope details could not be "
                                "found or fetched. Please ensure that the "
                                "correct scope name is provided.",
                        "httpStatusCode": 404,
                        "message": "Scope Not Found"
                    }
                else:
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = {
                        "code": 11001,
                        "hint": "The requested collection details could not "
                                "be found or fetched. Please ensure that the "
                                "correct collection name is provided.",
                        "httpStatusCode": 404,
                        "message": "Collection Not Found"
                    }`
	} else {
		elifs = ``
	}
	return elifs
}

func generatePythonFile(
	user string,
	commandUsed string,
	superClass string,
	nomenclature string,
	endpointParam string,
	endPoint Endpoint) error {

	// Create an __init__.py inside it to initialize the module.
	initFilePath := filepath.Join(destDir, "__init__.py")
	_, err := os.Create(initFilePath)
	if err != nil {
		return err
	}

	// Construct the destination file path
	destinationFilePath := filepath.Join(destDir, endPoint.fileName+".py")
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

	// Get the current date and format it.
	formattedDate := time.Now().Format("January 02, 2006")

	// Initialize other params which will be used further in the scripting
	url := endPoint.url
	successCodeArray := "[" + endPoint.statusCode + "]"
	v3endpoint := "/v3" + url[3:]
	pathParams := strings.Split(url, "/")
	lastPathParam := pathParams[len(pathParams)-1]
	invalidSegment := "/" + lastPathParam[:len(lastPathParam)-1]
	lastParamBad := ""
	for i, param := range strings.Split(url, "/") {
		if i == 0 {
			continue
		}
		if i == len(strings.Split(url, "/"))-1 {
			break
		}
		lastParamBad += "/" + param
	}
	lastParamBad += invalidSegment
	invalidSegmentError := `                "expected_status_code": 404,
                "expected_error": "404 page not found"`
	if endPoint.method == "LIST" {
		invalidSegmentError = `                "expected_status_code": 400,
                "expected_error": {
                    "code": 1000,
                    "hint": "Check if you have provided a valid URL and all "
                            "the required params are present in the request "
                            "body.",
                    "httpStatusCode": 400,
                    "message": "The server cannot or will not process the "
                               "request due to something that is perceived to "
                               "be a client error."
                }`
	} else if endPoint.method == "POST" {
		invalidSegmentError = `                "expected_status_code": 405,
                "expected_error": ""`
	}
	// Initialize formatted strings to be set up at the right place for function calls, eg,
	// normal calls have a 4 tab space indent but RATE LIMIT handled calls have a 5 tab space indent
	varTestPathParams := testPathParams(endPoint.parameters)
	var createParams string
	var authParam1 string
	var authParam2 string
	var authorizedRoles = "\n            "
	for i, role := range endPoint.authorizedRoles {
		if i > 0 && i < len(authorizedRoles)-1 {
			authorizedRoles += ","
			if i%3 == 0 {
				authorizedRoles += "\n            "
			}
		}
		authorizedRoles += " \"" + role + "\""
	}
	for i, param := range authorizationFuncCallParams {
		authParam1 += param
		authParam2 += param
		if i < len(authorizationFuncCallParams)-1 {
			createParams += param + ", "
			authParam1 += ", "
			authParam2 += ", "
			if i%2 == 0 && i > 0 {
				//authParam1 = authParam1[:len(authParam1)-1]
				authParam1 += "\n                "
				//authParam2 = authParam2[:len(authParam2)-1]
				authParam2 += "\n                    "
			}
		}
	}

	payloadParams = ""
	payloadParamsIndented = ""
	for _, param := range endPoint.requiredParameters {
		payloadParams += ",\n                self.expected_res[\"" + param + "\"]"
		payloadParamsIndented += ",\n                    self.expected_res[\"" + param + "\"]"
	}

	ifOrNotForValidation := ""
	if endPoint.method == "POST" || endPoint.method == "DELETE" {
		ifOrNotForValidation += "if "
	}

	// Write Python code to the file
	code := fmt.Sprintf(`"""
Created on %s

@author: Created using %s by %s
"""

%s


class %s(%s):%s

    def test_api_path(self):
        testcases = [
            {
                "description": "Send call with valid path params"
            }, {
                "description": "Replace api version in URI",
                "url": "%s",
                "expected_status_code": 404,
                "expected_error": {
                    "errorType": "RouteNotFound",
                    "message": "Not found"
                }
            }, {
                "description": "Replace the last path param name in URI",
                "url": "%s",
                "expected_status_code": 404,
                "expected_error": "404 page not found"
            }, {
                "description": "Add an invalid segment to the URI",
                "url": "%s%s",
%s
%s
        failures = list()
        for testcase in testcases:
            self.log.info("Executing test: {}".format(testcase["description"]))
%s
            if "url" in testcase:
                self.%s.%s = \
                    testcase["url"]
            if "invalid_organizationId" in testcase:
                organization = testcase["invalid_organizationId"]%s

            result = self.%s.%s(
                %s%s)
            if result.status_code == 429:
                self.handle_rate_limit(int(result.headers["Retry-After"]))
                result = self.%s.%s(
                    %s%s)
            self.%s.%s = \
                "%s"
            %sself.validate_testcase(result, %s, testcase, failures%s

        if failures:
            for fail in failures:
                self.log.warning(fail)
            self.fail("{} tests FAILED out of {} TOTAL tests"
                      .format(len(failures), len(testcases)))

    def test_authorization(self):
        failures = list()
        for testcase in self.v4_RBAC_injection_init([%s
        ]):
            self.log.info("Executing test: {}".format(testcase["description"]))
            header = dict()
            self.auth_test_setup(testcase, failures, header,
                                 self.project_id, self.other_project_id)
            result = self.%s.%s(
                %s%s,
                header)
            if result.status_code == 429:
                self.handle_rate_limit(int(result.headers["Retry-After"]))
                result = self.%s.%s(
                    %s%s,
                    header)
            %sself.validate_testcase(result, %s, testcase, failures%s

        if failures:
            for fail in failures:
                self.log.warning(fail)
            self.fail("{} tests FAILED.".format(len(failures)))
`, formattedDate, commandUsed, user, superClassImport, className, superClass, generateSetupAndTeardown(
		nomenclature, superClass, endPoint, createParams), v3endpoint, lastParamBad, url, invalidSegment,
		invalidSegmentError, fetchApiPathTestcasesFromSuperClass(endPoint.parameters), testApiPathParamInitialization,
		endPoint.api, endpointParam, varTestPathParams, endPoint.api, endPoint.funcName, apiPathFuncCallParams,
		payloadParams, endPoint.api, endPoint.funcName, apiPathFuncCallParams, payloadParamsIndented, endPoint.api,
		endpointParam, url, ifOrNotForValidation, successCodeArray, getExtraValidationParams(endPoint), authorizedRoles,
		endPoint.api, endPoint.funcName, authParam1, payloadParams, endPoint.api, endPoint.funcName, authParam2,
		payloadParamsIndented, ifOrNotForValidation, successCodeArray, getExtraValidationParams(endPoint))

	var strCombinations string
	var assignTestcase string
	var outermostIf string
	var firstInnerIf string
	var ifForCreateList string
	var typeCombinations string
	var testcaseParams1 string
	var testcaseParams2 string
	var correctParamsFormat string
	var createPathComb string
	for i := range combinationsParams {
		strCombinations += "str(" + combinationsParams[i][2] + "), "
		assignTestcase += "                \"" + combinationsParams[i][1] + "\": " + combinationsParams[i][2] + ",\n"

		outermostIf += combinationsParams[i][2] + " == " + authorizationFuncCallParams[i]
		if i < len(combinationsParams)-1 {
			outermostIf += " and\n                    "
		}

		if endPoint.method == "LIST" || endPoint.method == "POST" {
			if i != len(combinationsParams)-1 {
				firstInnerIf += combinationsParams[i][2] + ` == ""`
			}
			if i < len(combinationsParams)-2 {
				firstInnerIf += " or "
			}
		} else {
			firstInnerIf += combinationsParams[i][2] + ` == ""`
			if i != len(combinationsParams)-1 {
				firstInnerIf += " or "
			}
		}
		if i == 0 || i%2 == 0 {
			typeCombinations += "\n                             "
		}
		typeCombinations += "type(" + combinationsParams[i][2] + ")"
		if i < len(combinationsParams)-1 {
			typeCombinations += ", "
		}

		if i%2 == 0 {
			if i > 0 {
				testcaseParams1 = testcaseParams1[:len(testcaseParams1)-1]
				testcaseParams2 = testcaseParams2[:len(testcaseParams2)-1]
			}
			testcaseParams1 += "\n                "
			testcaseParams2 += "\n                    "
		}
		testcaseParams1 += "testcase[\"" + combinationsParams[i][1] + "\"], "
		testcaseParams2 += "testcase[\"" + combinationsParams[i][1] + "\"], "

		correctParamsFormat += authorizationFuncCallParams[i]
		createPathComb += authorizationFuncCallParams[i]

		if i != len(authorizationFuncCallParams)-1 {
			correctParamsFormat += ", "
			createPathComb += ", "
			if endPoint.method == "LIST" || endPoint.method == "POST" {
				if i%2 == 0 && i != 0 {
					firstInnerIf = firstInnerIf[:len(firstInnerIf)-1]
					firstInnerIf += "\n                        "
				}
			} else {
				if i%2 == 1 && i > 0 {
					firstInnerIf = firstInnerIf[:len(firstInnerIf)-1]
					firstInnerIf += "\n                        "
				}
			}
			if i%2 == 1 && i > 0 {
				strCombinations = strCombinations[:len(strCombinations)-1]
				strCombinations += "\n                        "
			}
			if i%3 == 2 && i > 0 {
				correctParamsFormat = correctParamsFormat[:len(correctParamsFormat)-1]
				correctParamsFormat += "\n                    "
				createPathComb = createPathComb[:len(createPathComb)-1]
				createPathComb += "\n                "
			}
		}
	}
	if testcaseParams1[len(testcaseParams1)-1] == ' ' || testcaseParams2[len(testcaseParams2)-1] == ' ' {
		testcaseParams1 = testcaseParams1[:len(testcaseParams1)-2]
		testcaseParams2 = testcaseParams2[:len(testcaseParams2)-2]
	}
	if endPoint.method == "POST" {
		ifForCreateList = fmt.Sprintf(`
                elif combination[%d] == "":
                    testcase["expected_status_code"] = 405
                    testcase["expected_error"] = ""
                elif `, len(combinationsParams)-1)
	} else if endPoint.method == "LIST" {
		ifForCreateList = fmt.Sprintf(`
                elif combination[%d] == "" or `, len(combinationsParams)-1)
	} else {
		ifForCreateList = `
                elif `
	}
	strCombinations = strCombinations[:len(strCombinations)-2]
	assignTestcase = assignTestcase[:len(assignTestcase)-2]

	code += fmt.Sprintf(`
    def test_query_parameters(self):
        self.log.debug(
                "Correct Params - %s".format(
                    %s))
        testcases = 0
        failures = list()
        for combination in self.create_path_combinations(
                %s):
            testcases += 1
            testcase = {
                "description": "%s"
                .format(%s),
%s
            }
            if not (%s):
                if (%s):
                    testcase["expected_status_code"] = 404
                    testcase["expected_error"] = "404 page not found"%sany(variable in [
                    int, bool, float, list, tuple, set, type(None)] for
                         variable in [%s]):
                    testcase["expected_status_code"] = 400
                    testcase["expected_error"] = {
                        "code": 1000,
                        "hint": "Check if you have provided a valid URL and "
                                "all the required params are present in the "
                                "request body.",
                        "httpStatusCode": 400,
                        "message": "The server cannot or will not process the "
                                   "request due to something that is "
                                   "perceived to be a client error."
                    }
                elif combination[0] != self.organisation_id:
                    testcase["expected_status_code"] = 403
                    testcase["expected_error"] = {
                        "code": 1002,
                        "hint": "Your access to the requested resource is "
                                "denied. Please make sure you have the "
                                "necessary permissions to access the "
                                "resource.",
                        "httpStatusCode": 403,
                        "message": "Access Denied."
                    }%s
            self.log.info("Executing test: {}".format(testcase["description"]))
            if "param" in testcase:
                kwarg = {testcase["param"]: testcase["paramValue"]}
            else:
                kwarg = dict()

            result = self.%s.%s(%s%s,
                **kwarg)
            if result.status_code == 429:
                self.handle_rate_limit(int(result.headers["Retry-After"]))
                result = self.%s.%s(%s%s,
                    **kwarg)
            %sself.validate_testcase(result, %s, testcase, failures%s

        if failures:
            for fail in failures:
                self.log.warning(fail)
            self.fail("{} tests FAILED out of {} TOTAL tests"
                      .format(len(failures), testcases))
`, correctQueryParams, correctParamsFormat, createPathComb, correctQueryParams, strCombinations, assignTestcase,
		outermostIf, firstInnerIf, ifForCreateList, typeCombinations, generateElifsForQueryTests(), endPoint.api,
		endPoint.funcName, authParam1, payloadParams, endPoint.api, endPoint.funcName, authParam2,
		payloadParamsIndented, ifOrNotForValidation, successCodeArray, getExtraValidationParams(endPoint))

	backtick := "`"
	payloadTestCode := fmt.Sprintf(`
        def test_payload(self):
        testcases = list()
        for k in self.expected_res:
            if k in []:
                continue

            for v in [
                "", 1, 0, 100000, -1, 123.123, None, [], {},
                self.generate_random_string(special_characters=False),
                self.generate_random_string(500, special_characters=False),
            ]:
                testcase = copy.deepcopy(self.expected_res)
                testcase[k] = v
                for param in []:
                    del testcase[param]
                testcase["description"] = "Testing %s{}%s with val: %s{}%s of {}"\
                    .format(k, v, type(v))
                # Add expected failure codes for malformed payload values...
                testcases.append(testcase)

        failures = list()
        for testcase in testcases:
            self.log.info(testcase['description'])
            result = self.%s.%s(
                %s%s)
            if result.status_code == 429:
                self.handle_rate_limit(int(result.headers["Retry-After"]))
                result = self.%s.%s(
                    %s%s)
            %sself.validate_testcase(result, %s, testcase, failures%s

        if failures:
            for fail in failures:
                self.log.warning(fail)
            self.fail("{} tests FAILED out of {} TOTAL tests"
                      .format(len(failures), len(testcases)))
`, backtick, backtick, backtick, backtick, endPoint.api, endPoint.funcName, testcaseParams1, payloadParams,
		endPoint.api, endPoint.funcName, testcaseParams2, payloadParamsIndented, ifOrNotForValidation, successCodeArray,
		getExtraValidationParams(endPoint))

	if endPoint.method == "POST" || endPoint.method == "PUT" {
		code += payloadTestCode
	}

	var rateLimitParams string
	for i, param := range authorizationFuncCallParams {
		rateLimitParams += param + ", "
		if i > 0 && i%2 == 0 && i != len(authorizationFuncCallParams)-1 {
			rateLimitParams = rateLimitParams[:len(rateLimitParams)-1]
			rateLimitParams += "\n                "
		}
	}
	//if endPoint.method == "PUT" || endPoint.method == "POST" {
	//	for _, param := range endPoint.payloadParams {
	//		rateLimitParams += param + ",\n                "
	//	}
	//	rateLimitParams += `
	//       ##################################################
	//       # Please add more params as necessary based on the type of request.
	//       # eg, for a CREATE or UPDATE request, add relevant params form
	//       # the self.expected_res object.
	//       ##################################################`
	//}
	rateLimitParams = rateLimitParams[:len(rateLimitParams)-2]

	// Throttle Tests (to check for rate limiting)
	code += fmt.Sprintf(`
    def test_multiple_requests_using_API_keys_with_same_role_which_has_access(
            self):
        api_func_list = [[
            self.%s.%s, (
                %s%s
            )
        ]]
        self.throttle_test(api_func_list)

    def test_multiple_requests_using_API_keys_with_diff_role(self):
        api_func_list = [[
            self.%s.%s, (
                %s%s
            )
        ]]
        self.throttle_test(api_func_list, True, self.project_id)
`, endPoint.api, endPoint.funcName, rateLimitParams, payloadParams, endPoint.api, endPoint.funcName, rateLimitParams,
		payloadParams)

	//strings.ReplaceAll(code, "\t", "    ")
	_, err = f.WriteString(code)
	if err != nil {
		return err
	}
	fmt.Printf("Python file '%s' generated successfully.\n", endPoint.fileName+".py")
	return nil
}

// cbFile represents the cbFile command
var cbFile = &cobra.Command{
	Use:   "cbFile",
	Short: "generates a REST API test automation script",
	Long: `cbFile: couchbase-RestAPI-Automation-Tool. 
The command can be used to create a python file inside a python package. 
Where the specified python file would test one of the specific pre-defined REST API endpoint.
Those pre-defined REST APIs are picked based on the flag parameter that is specified by the user.  
There are multiple REQUIRED flags that are mandatory for cbFile to have the complete information of what to be tested, and how can the script be syntactically the most accurate. 
Still it is advised to manually check the script once in case of anything that cbFile might miss as of now, since it is in the BETA stages. 

Usage example:
cbFile --nomenclature Alerts_List --superclass from pytests.Capella.RestAPIv4.Projects.get_projects import GetProject`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// Check in the MAP populated by cbRat.
		if pathsMap["scriptDir"] == "" {
			fmt.Println("No valid path found for the python test file to be generated.")
			fmt.Println("!!!...Please set the smokeUpgradeDir first using cbPaths...!!!")
		} else {
			destDir = pathsMap["scriptDir"]
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cbFile started...")

		user, _ := cmd.Flags().GetString("user")
		if user == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter the author name for the current automation tool user. " +
				"eg : Vipul Bhardwaj. (Person will be held responsible in case of test " +
				"failures once the script is generated and thoroughly checked and tests " +
				"are ran, verified, and the script code is pushed to GitHub) : ")
			user, _ = reader.ReadString('\n')
			user = strings.TrimSpace(user)
		}
		if user == "" {
			fmt.Println("err...Please provide a user who is the testing the current endpoint")
			return
		}

		nomenclature, _ := cmd.Flags().GetString("nomenclature")
		if nomenclature == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter nomenclature string for the current endpoint, " +
				"which will be used to name the Resources created in Capella while testing, " +
				"it goes as `ApiToBeTested_RestEndpointTypeThatItUses`: ")
			nomenclature, _ = reader.ReadString('\n')
			nomenclature = strings.TrimSpace(nomenclature)
		}
		if nomenclature == "" {
			fmt.Println("err...Please provide a nomenclature for the test endpoint")
			return
		}

		operationId, _ := cmd.Flags().GetString("operationId")
		if operationId == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter the operationId linked to the operation that is present in the YAML, " +
				"for which the API test script has to be automated : ")
			operationId, _ = reader.ReadString('\n')
			operationId = strings.TrimSpace(operationId)
		}
		if operationId == "" {
			fmt.Println("Please provide a valid operationId.")
			return
		}

		superClass, _ := cmd.Flags().GetString("superclass")
		if superClass == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter superclass which will be inherited by the current python test file being created, " +
				"it has the apiType first (Get, Create etc) and then the endpoint (Project, Bucket etc), " +
				"(eg - FooBar): ")
			superClass, _ = reader.ReadString('\n')
			superClass = strings.TrimSpace(superClass)
		}
		if superClass == "" {
			fmt.Println("Provide a valid existing class name for the test endpoint to use as a superclass: ")
			return
		}

		// Initializing other variables that have to be used in the test script.
		endPoint := specReader(pathsMap["readPath"], "", operationId)[0]
		commandUsed := cmd.CommandPath()
		className = toCamelCase(endPoint.method) + firstToUpper(endPoint.endpointParam)
		//className = className[:len(className)-1]
		choppedNames := strings.Split(nomenclature, "_")
		//apiType = choppedNames[1]
		endpoint = choppedNames[0]
		if endpoint[len(endpoint)-1] == 's' {
			endpoint = endpoint[:len(endpoint)-1]
		}
		resourceParam = strings.ToLower(endpoint) + "_id"
		//className = apiType + endpoint[:len(endpoint)-1]
		fmt.Printf("Class name generated for the test scipt: %s\n", className)

		superClassImport = "from pytests.Capella.RestAPIv4." + convertString(superClass) + "s import " + superClass
		fmt.Printf("Import string: %s\n", superClassImport)
		//fileName := strings.ToLower(apiType) + "_" + strings.ToLower(camelToSnake(endpoint)) + ".py"
		fmt.Printf("File name generated for the test scipt: %s\n", endPoint.fileName)
		//endpointParam := strings.ToLower(camelToSnake(endpoint)) + "_endpoint"

		// Calling the python script generator function
		if err := generatePythonFile(user, commandUsed, superClass, nomenclature, operationId,
			endPoint); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	},
}
