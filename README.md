# Custom Protocol Handler Launcher

A cross-platform application that handles custom URI schemes to open files in specific applications. For example, you can open remote files in Notepad or Photoshop directly from your browser using URIs like `launcher://notepad?url=https://example.com/file.txt`.

## Features

- Custom URI protocol handler (`launcher://`)
- Cross-platform support (Windows, macOS, Linux)
- Automatic file downloading and cleanup
- Configurable application paths
- Detailed logging system
- Support for multiple applications

## Prerequisites

- Go 1.16 or higher
- Administrator access (for installation)
- PowerShell (Windows)
- Supported applications (Notepad, Photoshop, etc.)

## Installation

1. Clone or download the source code
2. Navigate to the project directory
3. Build the application:
```bash
go build -o launcher
```

4. Install the protocol handler:
```bash
# Windows
.\launcher.exe -install

# macOS/Linux
./launcher -install
```

## Configuration

The application paths are defined in the `ApplicationPaths` variable. Modify these paths according to your system:

```go
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
```

## Usage

### Command Line

```bash
# Install protocol handler
launcher -install

# Uninstall protocol handler
launcher -uninstall

# Open a file directly
launcher -uri "launcher://notepad?url=https://example.com/file.txt"
```

### Browser Usage

After installation, you can use the launcher protocol in several ways:

1. Click links with the launcher protocol:
```html
<a href="launcher://notepad?url=https://example.com/file.txt">Open in Notepad</a>
```

2. Type in browser address bar:
```
launcher://notepad?url=https://example.com/file.txt
```

3. Use in JavaScript:
```javascript
window.location.href = "launcher://photoshop?url=https://example.com/image.jpg";
```

### Supported Applications

- Notepad (Windows) / TextEdit (macOS) / Gedit (Linux)
- Adobe Photoshop
- Add more by updating the `ApplicationPaths` variable

### URI Format

```
launcher://{application}?url={file_url}
```

Examples:
- `launcher://notepad?url=https://raw.githubusercontent.com/golang/go/master/README.md`
- `launcher://photoshop?url=https://example.com/image.jpg`

## Logging

Logs are stored in:
- Windows: `%USERPROFILE%\launcher_logs`
- macOS/Linux: `~/launcher_logs`

Log files are named by date: `launcher_YYYY-MM-DD.log`

## Troubleshooting

1. **Application doesn't open**: 
   - Check if the application path is correct in `ApplicationPaths`
   - Verify the application is installed
   - Check logs for detailed error messages

2. **Protocol not recognized**:
   - Reinstall the protocol handler: `launcher -install`
   - Verify registry entries (Windows) or bundle installation (macOS)

3. **File download fails**:
   - Verify the URL is accessible
   - Check network connectivity
   - Ensure write permissions in temp directory

## Uninstallation

```bash
# Windows
.\launcher.exe -uninstall

# macOS/Linux
./launcher -uninstall
```

## Security Considerations

- The application downloads files to a temporary location
- Files are automatically cleaned up after use
- Uses HTTPS for secure file downloads
- Installation is per-user (no admin rights required for usage)

## Contributing

Feel free to submit issues, fork the repository, and create pull requests for any improvements.

## License

[MIT License](LICENSE)