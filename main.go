package main

import (
	"encoding/json"
	"flag"
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

	"golang.org/x/sys/windows/registry"
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

// ApplicationInfo stores information about applications we want to find
type ApplicationInfo struct {
	Name           string
	WindowsRegKeys []string
	WindowsExeName string
	MacAppName     string
	LinuxBinName   string
}

// Define applications we want to search for
var applications = map[string]ApplicationInfo{
	"word": {
		Name: "Microsoft Word",
		WindowsRegKeys: []string{
			`SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\WINWORD.EXE`,
			`SOFTWARE\Microsoft\Office\Word\InstallRoot`,
		},
		WindowsExeName: "WINWORD.EXE",
		MacAppName:     "Microsoft Word.app",
		LinuxBinName:   "libreoffice",
	},
	"photoshop": {
		Name: "Adobe Photoshop",
		WindowsRegKeys: []string{
			`SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\Photoshop.exe`,
			`SOFTWARE\Adobe\Photoshop`,
		},
		WindowsExeName: "Photoshop.exe",
		MacAppName:     "Adobe Photoshop 2024.app",
		LinuxBinName:   "gimp",
	},
	"acrobat": {
		Name: "Adobe Acrobat",
		WindowsRegKeys: []string{
			`SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\Acrobat.exe`,
			`SOFTWARE\Adobe\Acrobat`,
		},
		WindowsExeName: "Acrobat.exe",
		MacAppName:     "Adobe Acrobat Reader.app",
		LinuxBinName:   "evince",
	},
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
		log.Fatal("Failed to get home directory:", err)
	}

	logDir := filepath.Join(homeDir, "hrep_launcher_logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatal("Failed to create log directory:", err)
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("launcher_%s.log", time.Now().Format("2006-01-02")))
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}

	log.SetOutput(f)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	return f
}

func main() {
	logFile := setupLogging()
	defer logFile.Close()

	installFlag := flag.Bool("install", false, "Install URI handler")
	uninstallFlag := flag.Bool("uninstall", false, "Uninstall URI handler")
	uriFlag := flag.String("uri", "", "URI to process")
	setPathFlag := flag.String("set-path", "", "Set custom path for an application (format: app=path)")
	listPathsFlag := flag.Bool("list-paths", false, "List all custom paths")
	removePathFlag := flag.String("remove-path", "", "Remove custom path for an application")
	setAssocFlag := flag.String("set-assoc", "", "Set custom file association (format: ext=app)")
	listAssocFlag := flag.Bool("list-assoc", false, "List all custom file associations")
	removeAssocFlag := flag.String("remove-assoc", "", "Remove custom file association for an extension")

	flag.Parse()

	// Load existing config
	config, err := loadConfig()
	if err != nil {
		config = Config{
			CustomPaths:            make(map[string]string),
			CustomFileAssociations: make(map[string]string),
		}
	}

	uri := *uriFlag
	if uri == "" && len(flag.Args()) > 0 {
		uri = strings.TrimPrefix(flag.Args()[0], "launcher://")
	}

	log.Printf("Launcher started. Install: %v, Uninstall: %v, URI: %s",
		*installFlag, *uninstallFlag, uri)

	switch {
	case *setPathFlag != "":
		parts := strings.SplitN(*setPathFlag, "=", 2)
		if len(parts) != 2 {
			log.Fatal("Invalid format for set-path. Use: -set-path app=path")
		}
		app, path := parts[0], parts[1]
		config.CustomPaths[app] = path
		if err := saveConfig(config); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}
		fmt.Printf("Custom path for %s set to: %s\n", app, path)
		// Rebuild application paths
		ApplicationPaths = buildApplicationMap()

	case *setAssocFlag != "":
		parts := strings.SplitN(*setAssocFlag, "=", 2)
		if len(parts) != 2 {
			log.Fatal("Invalid format for set-assoc. Use: -set-assoc .ext=app")
		}
		ext, app := parts[0], parts[1]
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		config.CustomFileAssociations[ext] = app
		if err := saveConfig(config); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}
		fmt.Printf("Custom association for %s set to: %s\n", ext, app)
		// Rebuild application paths
		ApplicationPaths = buildApplicationMap()

	case *listPathsFlag:
		fmt.Println("Custom application paths:")
		for app, path := range config.CustomPaths {
			fmt.Printf("%s: %s\n", app, path)
		}

	case *listAssocFlag:
		fmt.Println("Custom file associations:")
		for ext, app := range config.CustomFileAssociations {
			fmt.Printf("%s -> %s\n", ext, app)
		}

	case *removePathFlag != "":
		delete(config.CustomPaths, *removePathFlag)
		if err := saveConfig(config); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}
		fmt.Printf("Removed custom path for: %s\n", *removePathFlag)
		// Rebuild application paths
		ApplicationPaths = buildApplicationMap()

	case *removeAssocFlag != "":
		ext := *removeAssocFlag
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		delete(config.CustomFileAssociations, ext)
		if err := saveConfig(config); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}
		fmt.Printf("Removed custom association for: %s\n", ext)
		// Rebuild application paths
		ApplicationPaths = buildApplicationMap()

	case *installFlag:
		if err := installHandler(); err != nil {
			log.Fatalf("Installation failed: %v", err)
		}
		fmt.Println("URI handler installed successfully")

	case *uninstallFlag:
		if err := uninstallHandler(); err != nil {
			log.Fatalf("Uninstallation failed: %v", err)
		}
		fmt.Println("URI handler uninstalled successfully")

	case uri != "":
		if err := processURI(uri); err != nil {
			log.Fatalf("URI processing failed: %v", err)
		}

	default:
		flag.Usage()
	}
}

func findApplicationPaths() map[string]string {
	paths := make(map[string]string)

	// Load custom paths first
	config, err := loadConfig()
	if err == nil {
		for app, path := range config.CustomPaths {
			if fileExists(path) {
				paths[app] = path
				log.Printf("Loaded custom path for %s: %s", app, path)
			}
		}
	}

	// Then load system paths for apps that don't have custom paths
	switch runtime.GOOS {
	case "windows":
		systemPaths := findWindowsApplications()
		for app, path := range systemPaths {
			if _, exists := paths[app]; !exists {
				paths[app] = path
			}
		}
	case "darwin":
		systemPaths := findMacApplications()
		for app, path := range systemPaths {
			if _, exists := paths[app]; !exists {
				paths[app] = path
			}
		}
	case "linux":
		systemPaths := findLinuxApplications()
		for app, path := range systemPaths {
			if _, exists := paths[app]; !exists {
				paths[app] = path
			}
		}
	}

	return paths
}

func findWindowsApplications() map[string]string {
	paths := make(map[string]string)

	for appKey, appInfo := range applications {
		for _, regKey := range appInfo.WindowsRegKeys {
			if path := findWindowsAppPath(regKey, appInfo.WindowsExeName); path != "" {
				paths[appKey] = path
				break
			}
		}

		if paths[appKey] == "" {
			commonDirs := []string{
				`C:\Program Files`,
				`C:\Program Files (x86)`,
			}

			for _, dir := range commonDirs {
				if path := findWindowsAppInDir(dir, appInfo.WindowsExeName); path != "" {
					paths[appKey] = path
					break
				}
			}
		}
	}

	return paths
}

func findWindowsAppPath(regKeyPath, exeName string) string {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, regKeyPath, registry.READ)
	if err != nil {
		return ""
	}
	defer key.Close()

	path, _, err := key.GetStringValue("Path")
	if err == nil && path != "" {
		fullPath := filepath.Join(path, exeName)
		if fileExists(fullPath) {
			return fullPath
		}
	}

	return ""
}

func findWindowsAppInDir(rootDir, exeName string) string {
	var foundPath string
	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return filepath.SkipDir
		}
		if info.Name() == exeName {
			foundPath = path
			return filepath.SkipAll
		}
		return nil
	})
	return foundPath
}

func findMacApplications() map[string]string {
	paths := make(map[string]string)

	appDirs := []string{
		"/Applications",
		fmt.Sprintf("/Users/%s/Applications", os.Getenv("USER")),
	}

	for appKey, appInfo := range applications {
		for _, dir := range appDirs {
			path := filepath.Join(dir, appInfo.MacAppName)
			if fileExists(path) {
				paths[appKey] = path
				break
			}
		}
	}

	return paths
}

func findLinuxApplications() map[string]string {
	paths := make(map[string]string)

	binDirs := []string{
		"/usr/bin",
		"/usr/local/bin",
		"/opt",
	}

	for appKey, appInfo := range applications {
		for _, dir := range binDirs {
			path := filepath.Join(dir, appInfo.LinuxBinName)
			if fileExists(path) {
				paths[appKey] = path
				break
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

	parts := strings.SplitN(rawURI, "?url=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid URI format")
	}

	fileURL := parts[1]
	log.Printf("Full download URL: %s", fileURL)

	tempFile, err := downloadFile(fileURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	err = openFileWithApp(tempFile)
	if err != nil {
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
