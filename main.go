package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func prompt(question, defaultVal string) string {
	fmt.Printf("%s (default: %s): ", question, defaultVal)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return defaultVal
	}
	return input
}

func getCurrentProjectID() (string, error) {
	cmd := exec.Command("bash", "-c", "gcloud config get-value project 2>&1 | grep -v 'Your active configuration'")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current project ID: %v", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func getDefaultServiceAccount(projectID string) (string, error) {
	cmd := exec.Command("gcloud", "iam", "service-accounts", "list",
		"--filter=displayName:Compute Engine default service account",
		"--format=value(email)",
		"--project", projectID,
	)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get default service account: %v", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func main() {
	defaultZone := "europe-west1-b"
	defaultDiskSize := "50"

	currentProjectID, err := getCurrentProjectID()
	if err != nil {
		fmt.Println("Error getting current project ID:", err)
		return
	}

	projectID := prompt("Enter Project ID", currentProjectID)
	numVMsStr := prompt("Enter number of VMs to create", "")
	numVMs, err := strconv.Atoi(numVMsStr)
	if err != nil || numVMs <= 0 {
		fmt.Println("Invalid number of VMs.")
		return
	}

	vmNames := []string{}
	for i := 0; i < numVMs; i++ {
		name := prompt(fmt.Sprintf("Enter name for VM #%d", i+1), fmt.Sprintf("agent-%d", i+1))
		vmNames = append(vmNames, name)
	}

	storageSize := prompt("Enter storage size (GB)", defaultDiskSize)
	zone := prompt("Enter zone", defaultZone)

	serviceAccount, err := getDefaultServiceAccount(projectID)
	if err != nil {
		fmt.Println("Error getting default service account:", err)
		return
	}

	apply := prompt("Type 'apply' to run the commands, or anything else to just print them", "print")

	for _, name := range vmNames {
		command := []string{
			"compute", "instances", "create", name,
			"--project=" + projectID,
			"--zone=" + zone,
			"--machine-type=e2-standard-4",
			"--network-interface=network-tier=PREMIUM,stack-type=IPV4_ONLY,subnet=default",
			"--metadata=enable-osconfig=TRUE,enable-oslogin=true",
			"--maintenance-policy=MIGRATE",
			"--provisioning-model=STANDARD",
			"--service-account=" + serviceAccount,
			"--scopes=https://www.googleapis.com/auth/devstorage.read_only,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring.write,https://www.googleapis.com/auth/service.management.readonly,https://www.googleapis.com/auth/servicecontrol,https://www.googleapis.com/auth/trace.append",
			"--create-disk=auto-delete=yes,boot=yes,device-name=" + name + ",disk-resource-policy=projects/" + projectID + "/regions/europe-west1/resourcePolicies/default-schedule-1,image=projects/debian-cloud/global/images/debian-12-bookworm-v20250513,mode=rw,size=" + storageSize + ",type=pd-balanced",
			"--no-shielded-secure-boot",
			"--shielded-vtpm",
			"--shielded-integrity-monitoring",
			"--labels=goog-ops-agent-policy=v2-x86-template-1-4-0,goog-ec-src=vm_add-gcloud",
			"--reservation-affinity=any",
		}

		fmt.Println("\nRunning:", "gcloud", strings.Join(command, " "))
		if apply == "apply" {
			cmd := exec.Command("gcloud", command...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				fmt.Printf("Error creating VM %s: %v\n", name, err)
			}
		}
	}
}
