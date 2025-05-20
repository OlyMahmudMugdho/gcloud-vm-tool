package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

func prompt(promptText, defaultValue string) string {
	fmt.Printf("%s (default: %s): ", promptText, defaultValue)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

func getRegionFromZone(zone string) string {
	parts := strings.Split(zone, "-")
	if len(parts) >= 2 {
		return parts[0] + "-" + parts[1]
	}
	return zone
}

func runCommand(cmdStr string) error {
	fmt.Println("\nRunning:", cmdStr)
	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	projectID := prompt("Enter Project ID", "qwiklabs-gcp-02-091fb395ada9")
	numVMsStr := prompt("Enter number of VMs to create", "")
	var numVMs int
	fmt.Sscanf(numVMsStr, "%d", &numVMs)

	vmNames := make([]string, numVMs)
	for i := 0; i < numVMs; i++ {
		defaultName := fmt.Sprintf("agent-%d", i+1)
		vmNames[i] = prompt(fmt.Sprintf("Enter name for VM #%d", i+1), defaultName)
	}

	storageSize := prompt("Enter storage size (GB)", "50")
	zone := prompt("Enter zone", "europe-west1-b")
	region := getRegionFromZone(zone)

	mode := prompt("Type 'apply' to run the commands, or anything else to just print them", "print")

	// Step 1: Create firewall rule
	firewallCmd := fmt.Sprintf(
		"gcloud compute firewall-rules create allow-all "+
			"--network=default "+
			"--priority=101 "+
			"--direction=INGRESS "+
			"--action=ALLOW "+
			"--rules=all "+
			"--source-ranges=0.0.0.0/0 "+
			"--no-enable-logging "+
			"--project=%s", projectID)

	if mode == "apply" {
		fmt.Println("✅ Creating firewall rule...")
		if err := runCommand(firewallCmd); err != nil {
			fmt.Println("❌ Error creating firewall rule:", err)
		} else {
			fmt.Println("✅ Firewall rule created successfully")
		}
	} else {
		fmt.Println("🔧", firewallCmd)
	}

	// Step 2: Create VMs concurrently
	var wg sync.WaitGroup
	for _, vmName := range vmNames {
		vmName := vmName
		vmCmd := fmt.Sprintf(
			"gcloud compute instances create %s "+
				"--project=%s "+
				"--zone=%s "+
				"--machine-type=e2-standard-4 "+
				"--network-interface=network-tier=PREMIUM,stack-type=IPV4_ONLY,subnet=default "+
				"--metadata=enable-osconfig=TRUE,enable-oslogin=true "+
				"--maintenance-policy=MIGRATE "+
				"--provisioning-model=STANDARD "+
				"--service-account=%s@%s.iam.gserviceaccount.com "+
				"--scopes=https://www.googleapis.com/auth/devstorage.read_only,"+
				"https://www.googleapis.com/auth/logging.write,"+
				"https://www.googleapis.com/auth/monitoring.write,"+
				"https://www.googleapis.com/auth/service.management.readonly,"+
				"https://www.googleapis.com/auth/servicecontrol,"+
				"https://www.googleapis.com/auth/trace.append "+
				"--create-disk=auto-delete=yes,boot=yes,device-name=%s,"+
				"image=projects/debian-cloud/global/images/debian-12-bookworm-v20250513,"+
				"mode=rw,size=%s,type=pd-balanced,"+
				"disk-resource-policy=projects/%s/regions/%s/resourcePolicies/default-schedule-1 "+
				"--no-shielded-secure-boot "+
				"--shielded-vtpm "+
				"--shielded-integrity-monitoring "+
				"--labels=goog-ops-agent-policy=v2-x86-template-1-4-0,goog-ec-src=vm_add-gcloud "+
				"--reservation-affinity=any",
			vmName, projectID, zone,
			projectID, projectID,
			vmName, storageSize, projectID, region)

		if mode == "apply" {
			wg.Add(1)
			go func(cmd string, name string) {
				defer wg.Done()
				fmt.Printf("🚀 Creating VM %s...\n", name)
				if err := runCommand(cmd); err != nil {
					fmt.Printf("❌ Error creating VM %s: %v\n", name, err)
				} else {
					fmt.Printf("✅ VM %s created successfully\n", name)
				}
			}(vmCmd, vmName)
		} else {
			fmt.Println("🔧", vmCmd)
		}
	}

	if mode == "apply" {
		wg.Wait()
		fmt.Println("✅ All VM creation operations complete.")
	}
}
