package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func getRegionFromZone(zone string) string {
	parts := strings.Split(zone, "-")
	if len(parts) >= 2 {
		return parts[0] + "-" + parts[1] // e.g., us-west1-b => us-west1
	}
	return zone
}

func runCommand(cmd string) error {
	fmt.Println("Running:", cmd)
	out, err := exec.Command("bash", "-c", cmd).CombinedOutput()
	fmt.Println(string(out))
	return err
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Project ID (default: qwiklabs-gcp-02-091fb395ada9): ")
	projectID, _ := reader.ReadString('\n')
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		projectID = "qwiklabs-gcp-02-091fb395ada9"
	}

	fmt.Print("Enter number of VMs to create (default: 2): ")
	numVMsStr, _ := reader.ReadString('\n')
	numVMsStr = strings.TrimSpace(numVMsStr)
	numVMs := 2
	if numVMsStr != "" {
		fmt.Sscanf(numVMsStr, "%d", &numVMs)
	}

	// Default VM names will be agent-1, agent-2, ...
	vmNames := make([]string, numVMs)
	for i := 0; i < numVMs; i++ {
		defaultName := fmt.Sprintf("agent-%d", i+1)
		fmt.Printf("Enter name for VM #%d (default: %s): ", i+1, defaultName)
		vmName, _ := reader.ReadString('\n')
		vmName = strings.TrimSpace(vmName)
		if vmName == "" {
			vmName = defaultName
		}
		vmNames[i] = vmName
	}

	fmt.Print("Enter storage size (GB) (default: 50): ")
	storageSize, _ := reader.ReadString('\n')
	storageSize = strings.TrimSpace(storageSize)
	if storageSize == "" {
		storageSize = "50"
	}

	fmt.Print("Enter zone (default: europe-west1-b): ")
	zone, _ := reader.ReadString('\n')
	zone = strings.TrimSpace(zone)
	if zone == "" {
		zone = "europe-west1-b"
	}

	region := getRegionFromZone(zone)
	fmt.Printf("Extracted region: %s\n", region) // Display the region for clarity

	fmt.Print("Type 'apply' to run the commands, or anything else to just print them (default: print): ")
	action, _ := reader.ReadString('\n')
	action = strings.TrimSpace(action)
	if action == "" {
		action = "print"
	}

	// Example: Create a regional subnet (uses region)
	subnetCmd := fmt.Sprintf(`gcloud compute networks subnets create my-subnet --network=default --region=%s --range=10.0.0.0/24 --project=%s`, region, projectID)

	if action == "apply" {
		err := runCommand(subnetCmd)
		if err != nil {
			fmt.Println("❌ Error creating subnet:", err)
		} else {
			fmt.Println("✅ Subnet created successfully")
		}
	} else {
		fmt.Println("Subnet command:", subnetCmd)
	}

	// Firewall rule creation (firewall rules are global, no region needed)
	firewallCmd := fmt.Sprintf(`gcloud compute firewall-rules create allow-all --network=default --priority=101 --direction=INGRESS --action=ALLOW --rules=all --source-ranges=0.0.0.0/0 --no-enable-logging --project=%s`, projectID)

	if action == "apply" {
		err := runCommand(firewallCmd)
		if err != nil {
			fmt.Println("❌ Error creating firewall rule:", err)
		} else {
			fmt.Println("✅ Firewall rule created successfully")
		}
	} else {
		fmt.Println("Firewall command:", firewallCmd)
	}

	// Create VMs sequentially
	for _, vmName := range vmNames {
		vmCmd := fmt.Sprintf(
			`gcloud compute instances create %s --project=%s --zone=%s --machine-type=e2-standard-4 `+
				`--network-interface=network-tier=PREMIUM,stack-type=IPV4_ONLY,subnet=default `+
				`--metadata=enable-osconfig=TRUE,enable-oslogin=true --maintenance-policy=MIGRATE --provisioning-model=STANDARD `+
				`--service-account=%s@%s.iam.gserviceaccount.com `+
				`--scopes=https://www.googleapis.com/auth/devstorage.read_only,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring.write,https://www.googleapis.com/auth/service.management.readonly,https://www.googleapis.com/auth/servicecontrol,https://www.googleapis.com/auth/trace.append `+
				`--create-disk=auto-delete=yes,boot=yes,device-name=%s,image=projects/debian-cloud/global/images/debian-12-bookworm-v20250513,mode=rw,size=%s,type=pd-balanced `+
				`--no-shielded-secure-boot --shielded-vtpm --shielded-integrity-monitoring `+
				`--labels=goog-ops-agent-policy=v2-x86-template-1-4-0,goog-ec-src=vm_add-gcloud --reservation-affinity=any`,
			vmName, projectID, zone, projectID, projectID, vmName, storageSize,
		)

		if action == "apply" {
			fmt.Printf("🚀 Creating VM %s...\n\n", vmName)
			err := runCommand(vmCmd)
			if err != nil {
				fmt.Printf("❌ Error creating VM %s: %v\n", vmName, err)
			} else {
				fmt.Printf("✅ VM %s created successfully\n", vmName)
			}
		} else {
			fmt.Println("VM command:", vmCmd)
		}
	}

	fmt.Println("✅ All operations complete.")
}
