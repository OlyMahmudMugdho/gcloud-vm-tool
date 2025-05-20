package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

func prompt(label, defaultVal string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s (default: %s): ", label, defaultVal)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func getCurrentProjectID() (string, error) {
	cmd := exec.Command("gcloud", "config", "get-value", "project")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getDefaultServiceAccount(projectID string) (string, error) {
	cmd := exec.Command("gcloud", "iam", "service-accounts", "list", "--project", projectID, "--format=value(email)")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return "", fmt.Errorf("no service account found in project %s", projectID)
	}
	return lines[0], nil
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

	// --- Add firewall rule command ---
	firewallCmd := []string{
		"compute", "firewall-rules", "create", "allow-all",
		"--network=default",
		"--priority=101",
		"--direction=INGRESS",
		"--action=ALLOW",
		"--rules=all",
		"--source-ranges=0.0.0.0/0",
		"--no-enable-logging",
		"--project=" + projectID,
	}

	fmt.Printf("\nRunning: gcloud %s\n", strings.Join(firewallCmd, " "))
	if apply == "apply" {
		cmd := exec.Command("gcloud", firewallCmd...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println("⚠️ Failed to create firewall rule (may already exist):", err)
		} else {
			fmt.Println("✅ Firewall rule created successfully")
		}
	}

	// --- Concurrent VM creation ---
	var wg sync.WaitGroup
	for _, name := range vmNames {
		wg.Add(1)

		go func(vmName string) {
			defer wg.Done()

			command := []string{
				"compute", "instances", "create", vmName,
				"--project=" + projectID,
				"--zone=" + zone,
				"--machine-type=e2-standard-4",
				"--network-interface=network-tier=PREMIUM,stack-type=IPV4_ONLY,subnet=default",
				"--metadata=enable-osconfig=TRUE,enable-oslogin=true",
				"--maintenance-policy=MIGRATE",
				"--provisioning-model=STANDARD",
				"--service-account=" + serviceAccount,
				"--scopes=https://www.googleapis.com/auth/devstorage.read_only,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring.write,https://www.googleapis.com/auth/service.management.readonly,https://www.googleapis.com/auth/servicecontrol,https://www.googleapis.com/auth/trace.append",
				"--create-disk=auto-delete=yes,boot=yes,device-name=" + vmName + ",disk-resource-policy=projects/" + projectID + "/regions/europe-west1/resourcePolicies/default-schedule-1,image=projects/debian-cloud/global/images/debian-12-bookworm-v20250513,mode=rw,size=" + storageSize + ",type=pd-balanced",
				"--no-shielded-secure-boot",
				"--shielded-vtpm",
				"--shielded-integrity-monitoring",
				"--labels=goog-ops-agent-policy=v2-x86-template-1-4-0,goog-ec-src=vm_add-gcloud",
				"--reservation-affinity=any",
			}

			fmt.Printf("\nRunning: gcloud %s\n", strings.Join(command, " "))
			if apply == "apply" {
				cmd := exec.Command("gcloud", command...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					fmt.Printf("❌ Error creating VM %s: %v\n", vmName, err)
				} else {
					fmt.Printf("✅ VM %s created successfully\n", vmName)
				}
			}
		}(name)
	}

	wg.Wait()
	fmt.Println("\n✅ All VM creation operations complete.")
}
