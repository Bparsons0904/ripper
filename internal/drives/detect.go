package drives

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DriveInfo represents information about an optical drive
type DriveInfo struct {
	Device     string
	Model      string
	IsReadOnly bool
	MediaType  string
}

// DetectDrives scans for available optical drives on the system
func DetectDrives() ([]DriveInfo, error) {
	var drives []DriveInfo
	
	// Common device paths to check
	devicePaths := []string{
		"/dev/sr0", "/dev/sr1", "/dev/sr2", "/dev/sr3",
		"/dev/cdrom", "/dev/dvd", "/dev/cdrw",
	}
	
	for _, path := range devicePaths {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			drive := DriveInfo{
				Device:     path,
				Model:      getDeviceModel(path),
				IsReadOnly: isReadOnlyDevice(path),
				MediaType:  detectMediaType(path),
			}
			drives = append(drives, drive)
		}
	}
	
	// Also scan /sys/block for additional drives
	sysBlockDrives, err := scanSysBlock()
	if err == nil {
		drives = append(drives, sysBlockDrives...)
	}
	
	return drives, nil
}

// getDeviceModel attempts to get the model name of the device
func getDeviceModel(device string) string {
	// Extract device name (sr0, sr1, etc.)
	deviceName := filepath.Base(device)
	
	// Try to read model from /sys/block/*/device/model
	modelPath := fmt.Sprintf("/sys/block/%s/device/model", deviceName)
	if data, err := os.ReadFile(modelPath); err == nil {
		return strings.TrimSpace(string(data))
	}
	
	// Try vendor + model combination
	vendorPath := fmt.Sprintf("/sys/block/%s/device/vendor", deviceName)
	if vendorData, err := os.ReadFile(vendorPath); err == nil {
		vendor := strings.TrimSpace(string(vendorData))
		if modelData, err := os.ReadFile(modelPath); err == nil {
			model := strings.TrimSpace(string(modelData))
			return fmt.Sprintf("%s %s", vendor, model)
		}
		return vendor
	}
	
	return "Unknown Drive"
}

// isReadOnlyDevice checks if the device is read-only
func isReadOnlyDevice(device string) bool {
	deviceName := filepath.Base(device)
	roPath := fmt.Sprintf("/sys/block/%s/ro", deviceName)
	
	if data, err := os.ReadFile(roPath); err == nil {
		return strings.TrimSpace(string(data)) == "1"
	}
	
	return false
}

// detectMediaType attempts to detect what type of media is in the drive
func detectMediaType(device string) string {
	// Try to determine media type by checking device capabilities
	deviceName := filepath.Base(device)
	
	// Check if it's a DVD/BD capable drive
	capabilitiesPath := fmt.Sprintf("/proc/sys/dev/cdrom/info")
	if data, err := os.ReadFile(capabilitiesPath); err == nil {
		content := string(data)
		
		// Parse the capabilities file to determine drive type
		if strings.Contains(content, "DVD") {
			return "DVD/CD"
		}
		if strings.Contains(content, "BD") {
			return "Blu-ray/DVD/CD"
		}
	}
	
	// Fallback to CD for sr devices
	if strings.HasPrefix(deviceName, "sr") {
		return "CD/DVD"
	}
	
	return "Unknown"
}

// scanSysBlock scans /sys/block for optical drives
func scanSysBlock() ([]DriveInfo, error) {
	var drives []DriveInfo
	
	entries, err := os.ReadDir("/sys/block")
	if err != nil {
		return drives, err
	}
	
	// Look for sr* devices (SCSI CD-ROM)
	srPattern := regexp.MustCompile(`^sr\d+$`)
	
	for _, entry := range entries {
		if srPattern.MatchString(entry.Name()) {
			device := "/dev/" + entry.Name()
			
			// Only add if we haven't already found this device
			found := false
			for _, existing := range drives {
				if existing.Device == device {
					found = true
					break
				}
			}
			
			if !found {
				drive := DriveInfo{
					Device:     device,
					Model:      getDeviceModel(device),
					IsReadOnly: isReadOnlyDevice(device),
					MediaType:  detectMediaType(device),
				}
				drives = append(drives, drive)
			}
		}
	}
	
	return drives, nil
}

// HasMedia checks if there's media in the specified drive
func HasMedia(device string) bool {
	// Try to open the device to see if media is present
	file, err := os.Open(device)
	if err != nil {
		return false
	}
	defer file.Close()
	
	// If we can open it, there's likely media present
	// In a real implementation, you might want to use ioctl calls
	// to properly detect media presence
	return true
}

// GetPrimaryDrive returns the first available drive, or empty string if none found
func GetPrimaryDrive() string {
	drives, err := DetectDrives()
	if err != nil || len(drives) == 0 {
		return ""
	}
	
	return drives[0].Device
}