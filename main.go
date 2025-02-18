package main

import (
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
	"time"
)

const (
	appName = "launcher"
	version = "1.0.0"
)

// Set up logging
func init() {
	// Create logs directory in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Failed to get home directory:", err)
	}
	logDir := filepath.Join(homeDir, "launcher_logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatal("Failed to create log directory:", err)
	}

	// Set up log file
	logFile := filepath.Join(logDir, fmt.Sprintf("launcher_%s.log", time.Now().Format("2006-01-02")))
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}

	log.SetOutput(f)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

// ApplicationPaths stores the paths to applications for different operating systems
var ApplicationPaths = map[string]map[string]string{
	"windows": {
		"photoshop": "C:\\Program Files\\Adobe\\Adobe Photoshop CC 2024\\Photoshop.exe",
		"notepad":   "C:\\Windows\\notepad.exe",
	},
	"darwin": {
		"photoshop": "/Applications/Adobe Photoshop 2024/Adobe Photoshop 2024.app",
		"notepad":   "/System/Applications/TextEdit.app",
	},
	"linux": {
		"photoshop": "",
		"notepad":   "gedit",
	},
}

func main() {
	log.Println("Starting launcher application...")

	// Parse command line arguments
	install := flag.Bool("install", false, "Install the URI handler")
	uninstall := flag.Bool("uninstall", false, "Uninstall the URI handler")
	uri := flag.String("uri", "", "URI to handle")
	flag.Parse()

	// If no flags are provided, check for URI in remaining arguments
	// This handles the case when Windows calls the program with the URI directly
	if !*install && !*uninstall && *uri == "" && len(flag.Args()) > 0 {
		*uri = strings.TrimPrefix(flag.Args()[0], "launcher://")
	}

	log.Printf("Arguments: install=%v, uninstall=%v, uri=%s", *install, *uninstall, *uri)

	if *install {
		if err := installHandler(); err != nil {
			log.Fatal("Installation failed:", err)
		}
		fmt.Println("URI handler installed successfully!")
		time.Sleep(2 * time.Second) // Give user time to read message
	} else if *uninstall {
		if err := uninstallHandler(); err != nil {
			log.Fatal("Uninstallation failed:", err)
		}
		fmt.Println("URI handler uninstalled successfully!")
		time.Sleep(2 * time.Second)
	} else if *uri != "" {
		if err := handleURI(*uri); err != nil {
			log.Printf("Error handling URI: %v", err)
			time.Sleep(5 * time.Second) // Keep error message visible
			os.Exit(1)
		}
	} else {
		flag.Usage()
		time.Sleep(2 * time.Second)
	}
}

func handleURI(rawURI string) error {
	log.Printf("Handling URI: %s", rawURI)

	// If URI doesn't start with launcher://, add it
	if !strings.HasPrefix(rawURI, "launcher://") {
		rawURI = "launcher://" + rawURI
	}

	// Parse the URI
	parsedURI, err := url.Parse(rawURI)
	if err != nil {
		return fmt.Errorf("failed to parse URI: %w", err)
	}

	// Extract the application name and URL
	appName := strings.TrimPrefix(parsedURI.Host, "launcher://")
	fileURL := parsedURI.Query().Get("url")

	log.Printf("Application: %s, URL: %s", appName, fileURL)

	if fileURL == "" {
		return fmt.Errorf("no URL provided in the URI")
	}

	// Download the file
	tempFile, err := downloadFile(fileURL)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	// Launch application and wait for it to finish
	err = launchAndWait(appName, tempFile)

	// Clean up temp file after application closes
	os.Remove(tempFile)

	return err
}

func launchAndWait(appName, filePath string) error {
	os := runtime.GOOS
	appPaths, ok := ApplicationPaths[os]
	if !ok {
		return fmt.Errorf("unsupported operating system: %s", os)
	}

	appPath, ok := appPaths[appName]
	if !ok {
		return fmt.Errorf("unknown application: %s", appName)
	}

	var cmd *exec.Cmd
	switch os {
	case "windows":
		cmd = exec.Command(appPath, filePath)
	case "darwin":
		cmd = exec.Command("open", "-W", "-a", appPath, filePath)
	case "linux":
		cmd = exec.Command(appPath, filePath)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start application: %w", err)
	}

	return cmd.Wait()
}

func downloadFile(fileURL string) (string, error) {
	tempFile, err := os.CreateTemp("", "launcher-*"+filepath.Ext(fileURL))
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	resp, err := http.Get(fileURL)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return tempFile.Name(), nil
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
`, execPath)

	cmd := exec.Command("powershell", "-Command", regScript)
	return cmd.Run()
}

func installMacOS(execPath string) error {
	// Create app bundle directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	appDir := filepath.Join(homeDir, "Applications", "Launcher.app")
	contentsDir := filepath.Join(appDir, "Contents")
	macOSDir := filepath.Join(contentsDir, "MacOS")

	// Create directory structure
	dirs := []string{appDir, contentsDir, macOSDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Create Info.plist
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

	// Copy executable to MacOS directory
	binaryPath := filepath.Join(macOSDir, "launcher")
	if err := copyFile(execPath, binaryPath); err != nil {
		return err
	}

	// Make binary executable
	return os.Chmod(binaryPath, 0755)
}

func installLinux(execPath string) error {
	// Create desktop entry
	desktopEntry := fmt.Sprintf(`[Desktop Entry]
Name=Launcher
Exec=%s -uri %%u
Type=Application
Terminal=false
MimeType=x-scheme-handler/launcher;
`, execPath)

	// Save desktop entry
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

	// Register mime type
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
