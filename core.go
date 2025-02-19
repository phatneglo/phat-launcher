package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const (
	appName = "launcher"
	version = "1.0.0"
)

// Config stores all user configuration
type Config struct {
	CustomPaths            map[string]string `json:"customPaths"`
	CustomFileAssociations map[string]string `json:"customFileAssociations"`
}

// ApplicationPaths stores the discovered paths
var ApplicationPaths map[string]map[string]string

func init() {
	ApplicationPaths = buildApplicationMap()
}

// getConfigPath returns the path to the config file
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".launcher_config.json"), nil
}

// loadConfig loads configuration from file
func loadConfig() (Config, error) {
	config := Config{
		CustomPaths:            make(map[string]string),
		CustomFileAssociations: make(map[string]string),
	}

	configPath, err := getConfigPath()
	if err != nil {
		return config, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return config, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

// saveConfig saves configuration to file
func saveConfig(config Config) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func setupLogging() *os.File {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Write to temp dir if home dir is not accessible
		os.MkdirAll(os.TempDir()+"/hrep_launcher_logs", 0755)
		logFile := filepath.Join(os.TempDir(), "hrep_launcher_logs", fmt.Sprintf("launcher_%s.log", time.Now().Format("2006-01-02")))
		f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return nil
		}
		log.SetOutput(io.MultiWriter(f, os.Stderr))
		return f
	}

	logDir := filepath.Join(homeDir, "hrep_launcher_logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("launcher_%s.log", time.Now().Format("2006-01-02")))
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil
	}

	// Write to both file and stderr
	log.SetOutput(io.MultiWriter(f, os.Stderr))
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	return f
}

func findApplicationPaths() map[string]string {
	paths := make(map[string]string)

	// Load custom paths only
	config, err := loadConfig()
	if err == nil {
		for app, path := range config.CustomPaths {
			if fileExists(path) {
				paths[app] = path
			}
		}
	}

	return paths
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func processURI(rawURI string) error {
	log.Printf("Processing URI: %s", rawURI)

	// Remove launcher:// prefix if present
	rawURI = strings.TrimPrefix(rawURI, "launcher://")

	parts := strings.SplitN(rawURI, "?url=", 2)
	if len(parts) != 2 {
		log.Printf("Invalid URI format: missing ?url= parameter")
		return fmt.Errorf("invalid URI format")
	}

	fileURL := parts[1]
	log.Printf("Full download URL: %s", fileURL)

	tempFile, err := downloadFile(fileURL)
	if err != nil {
		log.Printf("Download failed: %v", err)
		return fmt.Errorf("download failed: %w", err)
	}
	log.Printf("Downloaded to temp file: %s", tempFile)

	// Get file extension and log application paths
	ext := strings.ToLower(filepath.Ext(tempFile))
	log.Printf("File extension: %s", ext)
	log.Printf("Available application paths: %+v", ApplicationPaths[runtime.GOOS])

	err = openFileWithApp(tempFile)
	if err != nil {
		log.Printf("Failed to open file: %v", err)
		os.Remove(tempFile)
		return err
	}

	return nil
}

func downloadFile(fileURL string) (string, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", fileURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %s", resp.Status)
	}

	parsedURL, err := url.Parse(fileURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	ext := filepath.Ext(parsedURL.Path)

	tempFile, err := os.CreateTemp("", fmt.Sprintf("hrep-launcher-*%s", ext))
	if err != nil {
		return "", fmt.Errorf("temp file creation failed: %w", err)
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("file write failed: %w", err)
	}

	return tempFile.Name(), nil
}

func openFileWithApp(filePath string) error {
	ext := strings.ToLower(filepath.Ext(filePath))
	os := runtime.GOOS

	appPath, exists := ApplicationPaths[os][ext]
	if !exists {
		return openWithDefaultApp(filePath)
	}

	var cmd *exec.Cmd
	switch os {
	case "windows":
		// Try to execute directly first
		cmd = exec.Command(appPath, filePath)
		err := cmd.Start()
		if err != nil {
			if strings.Contains(err.Error(), "requires elevation") {
				// If elevation is required, use 'runas' verb with ShellExecute
				cmd = exec.Command("cmd", "/c", "start", "/wait", "", appPath, filePath)
				cmd.SysProcAttr = &syscall.SysProcAttr{
					HideWindow:    true,
					CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
				}
			} else {
				return err
			}
		} else {
			return nil
		}
	case "darwin":
		cmd = exec.Command("open", "-a", appPath, filePath)
	case "linux":
		cmd = exec.Command(appPath, filePath)
	default:
		return fmt.Errorf("unsupported OS: %s", os)
	}

	log.Printf("Executing command: %v with file: %s", cmd.Path, filePath)
	return cmd.Start()
}

func openWithDefaultApp(filePath string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", filePath)
	case "darwin":
		cmd = exec.Command("open", filePath)
	case "linux":
		cmd = exec.Command("xdg-open", filePath)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	return cmd.Start()
}

func installHandler() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	switch runtime.GOOS {
	case "windows":
		return installWindows(execPath)
	case "darwin":
		return installMacOS(execPath)
	case "linux":
		return installLinux(execPath)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func installWindows(execPath string) error {
	regScript := fmt.Sprintf(`
$RegKey = "HKCU:\Software\Classes\launcher"
New-Item -Path $RegKey -Force
Set-ItemProperty -Path $RegKey -Name "(Default)" -Value "URL:Launcher Protocol"
Set-ItemProperty -Path $RegKey -Name "URL Protocol" -Value ""
New-Item -Path "$RegKey\shell\open\command" -Force
Set-ItemProperty -Path "$RegKey\shell\open\command" -Name "(Default)" -Value '"%s" "%%1"'

# Add runas verb for elevation when needed
New-Item -Path "$RegKey\shell\runas" -Force
Set-ItemProperty -Path "$RegKey\shell\runas" -Name "(Default)" -Value "Run as administrator"
New-Item -Path "$RegKey\shell\runas\command" -Force
Set-ItemProperty -Path "$RegKey\shell\runas\command" -Name "(Default)" -Value '"%s" "%%1"'
`, execPath, execPath)

	cmd := exec.Command("powershell", "-Command", regScript)
	return cmd.Run()
}

func installMacOS(execPath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	appDir := filepath.Join(homeDir, "Applications", "Launcher.app")
	contentsDir := filepath.Join(appDir, "Contents")
	macOSDir := filepath.Join(contentsDir, "MacOS")

	dirs := []string{appDir, contentsDir, macOSDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	infoPlist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleIdentifier</key>
    <string>com.example.launcher</string>
    <key>CFBundleName</key>
    <string>Launcher</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleExecutable</key>
    <string>launcher</string>
    <key>CFBundleVersion</key>
    <string>%s</string>
    <key>CFBundleURLTypes</key>
    <array>
        <dict>
            <key>CFBundleURLName</key>
            <string>Launcher Protocol</string>
            <key>CFBundleURLSchemes</key>
            <array>
                <string>launcher</string>
            </array>
        </dict>
    </array>
</dict>
</plist>`, version)

	if err := os.WriteFile(filepath.Join(contentsDir, "Info.plist"), []byte(infoPlist), 0644); err != nil {
		return err
	}

	binaryPath := filepath.Join(macOSDir, "launcher")
	if err := copyFile(execPath, binaryPath); err != nil {
		return err
	}

	return os.Chmod(binaryPath, 0755)
}

func installLinux(execPath string) error {
	desktopEntry := fmt.Sprintf(`[Desktop Entry]
Name=Launcher
Exec=%s -uri %%u
Type=Application
Terminal=false
MimeType=x-scheme-handler/launcher;
`, execPath)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	desktopFile := filepath.Join(homeDir, ".local", "share", "applications", "launcher.desktop")
	if err := os.MkdirAll(filepath.Dir(desktopFile), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(desktopFile, []byte(desktopEntry), 0644); err != nil {
		return err
	}

	cmd := exec.Command("xdg-mime", "default", "launcher.desktop", "x-scheme-handler/launcher")
	return cmd.Run()
}

func uninstallHandler() error {
	switch runtime.GOOS {
	case "windows":
		return uninstallWindows()
	case "darwin":
		return uninstallMacOS()
	case "linux":
		return uninstallLinux()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func uninstallWindows() error {
	cmd := exec.Command("powershell", "-Command", `Remove-Item -Path "HKCU:\Software\Classes\launcher" -Recurse -Force`)
	return cmd.Run()
}

func uninstallMacOS() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(homeDir, "Applications", "Launcher.app"))
}

func uninstallLinux() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return os.Remove(filepath.Join(homeDir, ".local", "share", "applications", "launcher.desktop"))
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0644)
}

func buildApplicationMap() map[string]map[string]string {
	appPaths := findApplicationPaths()
	extMap := make(map[string]map[string]string)

	// Load custom file associations
	config, err := loadConfig()
	if err == nil {
		for _, os := range []string{"windows", "darwin", "linux"} {
			extMap[os] = make(map[string]string)

			// Apply custom file associations first
			for ext, app := range config.CustomFileAssociations {
				if path, ok := appPaths[app]; ok {
					extMap[os][ext] = path
				}
			}

			// Then apply default associations for extensions that don't have custom ones
			if path, ok := appPaths["word"]; ok {
				if _, exists := extMap[os][".docx"]; !exists {
					extMap[os][".docx"] = path
				}
			}
			if path, ok := appPaths["photoshop"]; ok {
				for _, ext := range []string{".jpg", ".jpeg", ".png", ".psd"} {
					if _, exists := extMap[os][ext]; !exists {
						extMap[os][ext] = path
					}
				}
			}
			if path, ok := appPaths["acrobat"]; ok {
				if _, exists := extMap[os][".pdf"]; !exists {
					extMap[os][".pdf"] = path
				}
			}
		}
	}

	return extMap
}
