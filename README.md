# Launcher Application

A protocol handler application that allows opening files with specific applications based on file extensions.

## Prerequisites

- Go 1.23.3 or higher
- Windows, macOS, or Linux operating system
- Git (for cloning the repository)

## Project Setup

1. Clone the repository:
```bash
git clone https://github.com/phatneglo/phat-launcher.git HREP_GO
cd HREP_GO
```

2. Switch to the hrep_go branch:
```bash
git checkout hrep_go
```

3. Initialize go module and get dependencies:
```bash
# Initialize the module
go mod init launcher

# Get required dependencies
go get github.com/jchv/go-webview2@v0.0.0-20221223143126-dc24628cff85
```

4. Run go mod tidy to ensure all dependencies are properly synchronized:
```bash
go mod tidy
```

## Project Structure

Your project structure should look like this:

```
launcher/
├── .gitignore
├── core.go
├── go.mod
├── go.sum
├── Launcher GUI User Guide.md
├── main.go
└── README.md
```

## Building the Application

### Windows
```bash
go build -ldflags="-H windowsgui" -o launcher.exe
```
The `-H windowsgui` flag prevents the command prompt window from appearing when running the GUI application.

### macOS
```bash
go build -o launcher
```
On macOS, the application will run as a GUI application by default.

### Linux
```bash
go build -o launcher
```
On Linux, you might want to add the executable flag after building:
```bash
chmod +x launcher
```

## Running the application

After building the application, you can check [Launcher GUI User Guide](Launcher%20GUI%20User%20Guide.md) on how to use and install the application

## Usage

Once installed and configured, the launcher can handle URLs with the following format:
```
launcher://?url=https://example.com/path/to/file.pdf
```

You can use this on html like this:
```html
   <a href="launcher://?url=https://example.com/path/to/file.pdf">Open File</a>
```

## Command Line Options

```
-cli           Run in CLI mode
-install       Install URI handler
-uninstall     Uninstall URI handler
-uri           URI to process
-set-path      Set custom path for an application (format: app=path)
-list-paths    List all custom paths
-remove-path   Remove custom path for an application
-set-assoc     Set custom file association (format: ext=app)
-list-assoc    List all custom file associations
-remove-assoc  Remove custom file association for an extension
```

## Development

### Project Files Overview

- `main.go`: Contains the main application logic, GUI implementation, and CLI handling
- `core.go`: Contains core functionality for handling file operations, configurations, and protocol handling

### Adding New Features

To add support for new file types:
1. Add the file extension to the `buildApplicationMap` function in `core.go`
2. Configure the application path using the GUI or CLI
3. Test the new file association

## Troubleshooting

### Common Issues

1. **WebView2 Runtime Requirements**

   a. **Check WebView2 Installation**
   - Open Windows Registry Editor (regedit)
   - Navigate to: `HKEY_LOCAL_MACHINE\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`
   - If this key exists, WebView2 Runtime is installed
   - Alternatively, check `C:\Program Files (x86)\Microsoft\EdgeWebView\Application` for installation files

   b. **Install WebView2 Runtime**
   - Download the WebView2 Evergreen Bootstrapper from: https://developer.microsoft.com/en-us/microsoft-edge/webview2/
   - Choose "Evergreen Bootstrapper" (not the fixed version)
   - Direct download link: https://go.microsoft.com/fwlink/p/?LinkId=2124703
   - Run installer with administrator privileges
   - The bootstrapper will automatically install the correct version for your system

   c. **System Requirements**
   - Windows 10 version 1803 (OS Build 17134) or later
   - Windows 11
   - Windows Server 2019 or later
   - x86, x64, or ARM64 processor architecture

   d. **Dependencies Check**
   - Microsoft Visual C++ Redistributable required
   - .NET Framework 4.6.2 or later recommended
   - Check Windows Updates are current

   e. **Troubleshooting Installation**
   - Run `PowerShell` as administrator
   - Check version: 
     ```powershell
     Get-ItemProperty -Path 'HKLM:\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}' -Name pv
     ```
   - If installation fails, try:
     1. Uninstall existing WebView2 Runtime
     2. Clear temp files in `%TEMP%`
     3. Restart computer
     4. Install Evergreen Bootstrapper again with admin rights

2. **Permission Issues**
   - On Windows, run with administrator privileges for installation
   - On Linux/macOS, ensure proper file permissions

3. **Configuration Not Saving**
   - Check write permissions in the user's home directory
   - Verify the config file location (~/.launcher_config.json)

### Logs

Logs are stored in:
- Windows: `%USERPROFILE%\hrep_launcher_logs`
- macOS/Linux: `~/hrep_launcher_logs`
