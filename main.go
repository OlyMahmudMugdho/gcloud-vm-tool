package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func getCurrentProjectID() string {
	cmd := exec.Command("bash", "-c", `gcloud config get-value project 2>&1 | grep -v "Your active configuration"`)
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error fetching current project ID:", err)
		os.Exit(1)
	}
	return strings.TrimSpace(string(output))
}

func prompt(question, defaultValue string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s (default: %s): ", question, defaultValue)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

func main() {
	currentProject := getCurrentProjectID()
	projectID := prompt("Enter Project ID", currentProject)

	numVMsStr := prompt("Enter number of VMs to create", "")
	var numVMs int
	fmt.Sscanf(numVMsStr, "%d", &numVMs)

	vmNames := make([]string, numVMs)
	for i := 0; i < numVMs; i++ {
		vmNames[i] = prompt(fmt.Sprintf("Enter name for VM #%d", i+1), fmt.Sprintf("agent-%d", i+1))
	}

	storageSize := prompt("Enter storage size (GB)", "50")
	zone := prompt("Enter zone", "europe-west1-b")

	apply := prompt("Type 'apply' to run the commands, or anything else to just print them", "print")

	for _, name := range vmNames {
		createCmd := fmt.Sprintf(`gcloud compute instances create %s --project=%s --zone=%s --machine-type=e2-standard-4 \
--network-interface=network-tier=PREMIUM,stack-type=IPV4_ONLY,subnet=default \
--metadata=enable-osconfig=TRUE,enable-oslogin=true \
--maintenance-policy=MIGRATE --provisioning-model=STANDARD \
--service-account=YOUR_SERVICE_ACCOUNT_EMAIL \
--scopes=https://www.googleapis.com/auth/devstorage.read_only,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring.write,https://www.googleapis.com/auth/service.management.readonly,https://www.googleapis.com/auth/servicecontrol,https://www.googleapis.com/auth/trace.append \
--create-disk=auto-delete=yes,boot=yes,device-name=%s,disk-resource-policy=projects/%s/regions/%s/resourcePolicies/default-schedule-1,image=projects/debian-cloud/global/images/debian-12-bookworm-v20250513,mode=rw,size=%s,type=pd-balanced \
--no-shielded-secure-boot --shielded-vtpm --shielded-integrity-monitoring \
--labels=goog-ops-agent-policy=v2-x86-template-1-4-0,goog-ec-src=vm_add-gcloud --reservation-affinity=any`, name, projectID, zone, name, projectID, strings.TrimSuffix(zone, "-b"), storageSize)

		if apply == "apply" {
			cmd := exec.Command("bash", "-c", createCmd)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			fmt.Println("\nRunning:", createCmd)
			cmd.Run()
		} else {
			fmt.Println("\nCommand to run:")
			fmt.Println(createCmd)
		}
	}
}
