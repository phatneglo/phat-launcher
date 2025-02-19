# LIRMDS Client Launcher - GUI User Guide

## Getting Started

1. Run `launcher.exe` without any command-line arguments to open the GUI interface or simply click the launcher.exe
2. The interface will open in a window titled "LIRMDS Client Launcher Configuration"

## Interface Sections

### 1. Application Paths

This section allows you to configure paths to various applications that will handle different file types.

#### Adding a New Application Path:
1. In the "Application Paths" section:
   - Enter the application name (e.g., "word", "acrobat", "photoshop")
   - Click "Browse" to select the application's executable file
   - Click "Add Path" to save

Example applications:
- `word`: Path to Microsoft Word executable
- `acrobat`: Path to Adobe Acrobat
- `photoshop`: Path to Adobe Photoshop

#### Removing an Application Path:
- Click the "Remove" button next to the application path you want to delete
- Confirm the removal when prompted

### 2. File Associations

This section lets you associate file extensions with applications you've configured.

#### Adding a File Association:
1. Enter the file extension (e.g., ".pdf", ".docx", ".jpg")
   - Make sure to include the dot (.)
2. Select the application from the dropdown menu
3. Click "Add Association"

Common associations:
- `.pdf` → acrobat
- `.docx` → word
- `.jpg`, `.png`, `.psd` → photoshop

#### Removing a File Association:
- Click the "Remove" button next to the association you want to delete
- Confirm the removal when prompted

### 3. Protocol Handler Installation

This section manages the system integration of the launcher.

#### Installing the Protocol Handler:
1. Click "Install Protocol Handler"
2. Wait for the success message
3. The launcher will now handle `launcher://` URLs

#### Uninstalling the Protocol Handler:
1. Click "Uninstall Protocol Handler"
2. Wait for the success message
3. The launcher will no longer handle `launcher://` URLs

## Configuration Storage

- All configurations are automatically saved to `%USERPROFILE%\.launcher_config.json`
- The file is created automatically when you first add a configuration
- Backup this file if you want to preserve your settings

## Testing Your Configuration

1. After setting up application paths and file associations, you can test the configuration:
   - The launcher will handle URLs in the format: `launcher://open?url=https://example.com/path/to/file.pdf`
   - Files will open in their associated applications

## Troubleshooting

### Common Issues

1. **Application Not Found**
   - Verify the application path exists
   - Check if the path contains spaces and is properly configured
   - Try removing and re-adding the application path

2. **File Association Not Working**
   - Ensure the file extension includes the dot (.)
   - Verify the associated application is properly configured
   - Try removing and re-adding the association

3. **Protocol Handler Issues**
   - Try uninstalling and reinstalling the protocol handler
   - Run launcher.exe as administrator when installing the protocol handler
   - Check if WebView2 Runtime is properly installed

### Logs

If you encounter issues, check the log files located at:
- `%USERPROFILE%\hrep_launcher_logs\launcher_YYYY-MM-DD.log`

## Best Practices

1. **Application Names**
   - Use simple, lowercase names without spaces
   - Stick to common names like "word", "acrobat", "photoshop"

2. **File Extensions**
   - Always include the dot (.)
   - Use lowercase for extensions
   - Example: ".pdf", ".docx", ".jpg"

3. **Regular Maintenance**
   - Periodically verify application paths still exist
   - Remove unused associations
   - Keep the WebView2 Runtime updated

## Security Notes

1. Always download applications from trusted sources
2. Verify executable paths when browsing
3. Run as administrator only when necessary for installation

## Updating the Launcher

1. Back up your configuration file
2. Install the new version
3. Reinstall the protocol handler
4. Verify your configurations are intact

Remember to check the logs if you encounter any issues during usage. Logs provide detailed information about what the launcher is doing and can help identify problems.