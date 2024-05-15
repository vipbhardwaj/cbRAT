package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var regionMap = map[string]string{
	"AWS":   "us-east-1",
	"Azure": "eastus",
	"GCP":   "us-east1",
}

// cbSmokeUpgrade represents the cbSmokeUpgrade command
var cbSmokeUpgrade = &cobra.Command{
	Use:   "cbSmokeUpgrade",
	Short: "generates a smoke upgrade YAML file in CP-CLI for a specific AMI to be tested on cloud.",
	Long: `cbSmokeUpgrade:
The command can be used to create a YAML file in cp-cli to which is used for smoke upgrade testing for a specific AMI for a cloud provider.
The added code follows just a fixed steps called 'actions' which are bound in trees, and these follow test guidelines as of April 2024.
Some fixed parameters such as the load that has to be applied while upgrading / scaling up a cluster can be subject to change, which is currently fixed to a specific threshold. 
Make sure to provide the 4 essential parameters : 
	- The cloud provider
	- The version for the cluster to be deployed (upgrading from)
	- The Image that is associated with that cluster version
	- The releaseId for the image related to the version
All these 4 params can be passed through mandatory flags that are required while using cbSmokeUpgrade.

Usage example:
cbSmokeUpgrade --cloud AWS --version "7.2.3" --image "couchbase-cloud-server-7.2.3-6705-x86_64-v1.0.24" --releaseId "1.0.24"`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// Check in the MAP populated by cbRat.
		if pathsMap["smokeUpgradeDir"] == "" {
			fmt.Println("No valid path found for the smoke upgrade file to be generated.")
			fmt.Println("!!!...Please set the smokeUpgradeDir first using cbPaths...!!!")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cbSmokeUpgrade started...")

		cloud, _ := cmd.Flags().GetString("cloud")
		if cloud == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter the cloud provider for which the cluster would be deployed, (it is case sensitive) : ")
			cloud, _ = reader.ReadString('\n')
			cloud = strings.TrimSpace(cloud)
		}
		if cloud != "AWS" && cloud != "GCP" && cloud != "Azure" {
			fmt.Println("Please provide a valid cloud for which the cluster will be deployed...!")
			return
		}

		//version, _ := cmd.Flags().GetString("version")
		//if version == "" {
		//	reader := bufio.NewReader(os.Stdin)
		//	fmt.Print("Enter the cluster version to be deployed : ")
		//	version, _ = reader.ReadString('\n')
		//	version = strings.TrimSpace(version)
		//}
		//if version == "" || len(strings.Split(version, ".")) != 3 {
		//	fmt.Println("Please provide a valid version for the cluster, eg 'x.y.z'...!")
		//	return
		//}

		image, _ := cmd.Flags().GetString("image")
		if image == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter the cluster version to be deployed : ")
			image, _ = reader.ReadString('\n')
			image = strings.TrimSpace(image)
		}
		if image == "" {
			fmt.Println("Please provide a valid image for the cluster...!")
			return
		}

		var version string
		temp := strings.Split(image, "couchbase-cloud-server-")[1]
		for i := range 5 {
			if temp[i] == '-' {
				version += "."
			} else {
				version += string(temp[i])
			}
		}

		releaseId, _ := cmd.Flags().GetString("releaseId")
		if releaseId == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter the cluster releaseId related to the image : ")
			releaseId, _ = reader.ReadString('\n')
			releaseId = strings.TrimSpace(releaseId)
		}
		if releaseId == "" || len(strings.Split(releaseId, ".")) != 3 {
			fmt.Println("Please provide a valid releaseId for the image, eg '1.0.99'...!")
			return
		}

		//commandUsed := cmd.CommandPath()

		//getAmiParams(cloud, releaseIdString)

		// Calling the YAML script generator function
		if err := generateSmokeUpgradeYAML(cloud, regionMap[cloud], version, image, releaseId); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	},
}

//func getAmiParams(cloud string, releaseIdString string) (version string, image string, releaseId string) {
//	// Replace this URL with the raw URL of the file on GitHub
//	url := "https://raw.githubusercontent.com/username/repository/branch/path/to/file.txt"
//
//	// Make an HTTP GET request to fetch the file content
//	resp, err := http.Get(url)
//	if err != nil {
//		fmt.Printf("error fetching file: %s\n", err)
//		return
//	}
//	defer resp.Body.Close()
//
//	// Read the response body
//	body, err := ioutil.ReadAll(resp.Body)
//	if err != nil {
//		fmt.Printf("error reading response body: %s\n", err)
//		return
//	}
//
//	// Split the content into lines
//	lines := strings.Split(string(body), "\n")
//
//	// Iterate through each line
//	for _, line := range lines {
//		if strings.Split(strings.TrimSpace(line), releaseIdString)[0] == releaseIdString {
//
//		}
//	}
//}

func generateSmokeUpgradeYAML(cloud string, region string, version string, image string, releaseId string) error {
	// Construct the destination file path
	releaseId = strings.Replace(releaseId, ".", "", -1)
	fileName := fmt.Sprintf(`trinity-upgrade-%s.yaml`, releaseId)
	destinationFilePath := filepath.Join(pathsMap["smokeUpgradeDir"]+strings.ToLower(cloud)+"/", fileName)
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

	var code = fmt.Sprintf(`# serverImage specified in the Upgrade G2 Cluster action should be different from the deployed image.
action: Get JWT
iterations: 1
maxConcurrent: 5
timeoutInMins: 500
trees:
  - action: Deploy Project
    trees:
      - action: Deploy G2 Cluster
        config:
          provider: hosted%s
          region: %s
          templates: ["g2/quick-3.json"] # start off with 3 nodes
          serverVersion: "%s" # optional field. If not passed default version gets deployed
          serverImage: "%s" # optional field. If not passed default image gets deployed
          clusterName: "%s"
        trees:
          - action: Allow IP # allow ip and start pillowfight
            trees:
              - action: Create Bucket #Initial bucket creation
                config:
                  bucket: default1
                  bucketConflictResolution: seqno
                  memoryAllocationInMb: 2050
                  replicas: 2
                  flush: true
                  durabilityLevel: majority
                  storageBackend: magma
                trees: # create scopes and collections
                  - action: Run Container
                    config:
                      imageName: sequoiatools/collections:capella
                      params: -i <hostname>:18091 -u <username> -p <password> -b <bucketName>  -o create_multi_scope_collection -s scope_ -c coll_ --scope_count=2 --collection_count=10 --collection_distribution=uniform --tls True
                      bucketName: default1
                    trees:
                      - action: Run Container # create GSI indexes on all buckets
                        config:
                          imageName: sequoiatools/indexmanager
                          params: -n <hostname> -u <username> -p <password> --bucket_list default1 -a create_n_indexes_on_buckets --num_of_indexes_per_bucket 39 -x True
                          bucketName: default1
                        trees:
                          - action: Run Container # Create FTS indexes
                            config:
                              imageName: sequoiatools/ftsindexmanager
                              params: -n <hostname> -o 18091 -u <username> -p <password> -b <bucketName> -m 1:2:2,1:2:4 -s 1 -a create_index_from_map_on_bucket -tls True
                              bucketName: default1
                              bucketMemory: 2133
                            trees:
                              - action: Create Analytics Entities
                                config:
                                  numberOfDataverses: 30
                                  numberOfDatasets: 25
                                  numberOfSynonyms: 10
                                trees:
                                  - action: Create Function
                                    config:
                                      functionFile: eventing-functions/sample-eventing-function-backup-restore-tests/func_1.json
                                  - action: Create Function
                                    config:
                                      functionFile: eventing-functions/sample-eventing-function-backup-restore-tests/func_2.json
                                  - action: Create Function
                                    config:
                                      functionFile: eventing-functions/sample-eventing-function-backup-restore-tests/func_3.json
                                  - action: Create Function
                                    config:
                                      functionFile: eventing-functions/sample-eventing-function-backup-restore-tests/func_4.json
                                  - action: Create Function
                                    config:
                                      functionFile: eventing-functions/sample-eventing-function-backup-restore-tests/func_5.json
                                    trees:
                                      - action: Run Docloader
                                        config:
                                          params: -n <connection string> -user <username> -pwd <password> -b <bucket> -scope <scope> -collection <collection> -p 11207 -create_s 0 -create_e 1000000 -cr 100 -ops 100000 -docSize 1024 -workers 10
                                          bucket: default1
                                          scope: scope_0
                                          collection: coll_0
                                        trees:
                                          - action: Run Docloader
                                            config:
                                              params: -n <connection string> -user <username> -pwd <password> -b <bucket> -scope <scope> -collection <collection> -p 11207 -create_s 0 -create_e 1000000 -cr 100 -ops 100000 -docSize 1024 -workers 10
                                              bucket: default1
                                              scope: scope_0
                                              collection: coll_1
                                            trees:
                                              - action: Run Docloader
                                                config:
                                                  params: -n <connection string> -user <username> -pwd <password> -b <bucket> -scope <scope> -collection <collection> -p 11207 -create_s 0 -create_e 1000000 -cr 100 -ops 100000 -docSize 1024 -workers 10
                                                  bucket: default1
                                                  scope: scope_0
                                                  collection: coll_2
                                                trees:
                                                  - action: Run Docloader
                                                    config:
                                                      params: -n <connection string> -user <username> -pwd <password> -b <bucket> -scope <scope> -collection <collection> -p 11207 -create_s 0 -create_e 1000000 -cr 100 -ops 100000 -docSize 1024 -workers 10
                                                      bucket: default1
                                                      scope: scope_0
                                                      collection: coll_3
                                                    trees:
                                                      - action: Run Docloader
                                                        config:
                                                          params: -n <connection string> -user <username> -pwd <password> -b <bucket> -scope <scope> -collection <collection> -p 11207 -create_s 0 -create_e 1000000 -cr 100 -ops 100000 -docSize 1024 -workers 10
                                                          bucket: default1
                                                          scope: scope_0
                                                          collection: coll_4
                                                        trees:
                                                          - action: Start Pillowfight
                                                            config:
                                                              windowMins: 5
                                                              flags:
                                                                - "--num-threads=5"
                                                                - "--max-size=500"
                                                                - "--num-items=150"
                                                                - "--num-cycles=-1"
                                                                - "--batch-size=100"
                                                                - "--timings"
                                                                - "-Doperation_timeout=300"
                                                            trees:
                                                              - action: Modify G2 Cluster Specs
                                                                config:
                                                                  template: "g2/quick-5.json" # scale it out to 5 nodes
                                                                  scaleType: Scale-Out
                                                                  timeoutInMins: 200
                                                                trees:
                                                                  - action: Checklist
                                                                    config:
                                                                      provider: hostedAWS
                                                                      checks:
                                                                        - N1QLQueries
                                                                        - DeleteIndex
                                                                        - ImportSampleBucket
                                                                    trees:
                                                                      - action: Wait Time # Adding a wait period for the instance to become healthy after AZ failure and node termination
                                                                        config:
                                                                          sleepTimeMins: 2
                                                                        trees:
                                                                          - action: Upgrade G2 Cluster
                                                                            config:
                                                                              releaseId: "1.0.25"
                                                                              serverImage: "couchbase-cloud-server-7.6.0-2090-x86_64-v1.0.28"
                                                                              serverVersion: "7.6.0"
                                                                              timeoutInMins: 200
                                                                            trees:
                                                                              - action: Checklist
                                                                                config:
                                                                                  provider: hostedAWS
                                                                                  checks:
                                                                                    - CreateFTS
                                                                                    - CheckBucketsHealth
                                                                                    - N1QLQueries
                                                                                    - DeleteIndex
                                                                                trees:
                                                                                  - action: Modify G2 Cluster Specs
                                                                                    config:
                                                                                      template: "g2/quick-3.json" # scale it in to 3 nodes
                                                                                      scaleType: Scale-In
                                                                                      timeoutInMins: 200
                                                                                    trees:
                                                                                      - action: Checklist
                                                                                        config:
                                                                                          provider: hostedAWS
                                                                                          checks:
                                                                                            - N1QLQueries
                                                                                            - DeleteIndex
                                                                                            - CheckRebalance
                                                                                            - ReceiveMetrics
                                                                                        trees:
                                                                                          - action: Stop Pillowfight
                                                                                            trees:
                                                                                              - action: Destroy G2 Cluster
                                                                                                trees:
                                                                                                  - action: Wait Time # Adding a wait period for other two jobs to finish too.
                                                                                                    config:
                                                                                                      sleepTimeMins: 40
`, cloud, region, version, image, releaseId)

	_, err = f.WriteString(code)
	if err != nil {
		return err
	}
	fmt.Printf("Python file '%s' generated successfully.\n", fileName)
	return nil
}
