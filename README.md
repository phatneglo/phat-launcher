# Launcher Application Documentation

## Overview
The Launcher is a protocol handler application that manages file associations and opens files with specific applications. It supports custom paths for portable applications and allows custom file type associations.

## Quick Start

1. Download `launcher.exe`
2. Open command prompt as administrator
3. Install the protocol handler:
```bash
launcher.exe -install
```

## Command Reference

### Installation Commands

```bash
# Install the protocol handler
launcher.exe -install

# Uninstall the protocol handler
launcher.exe -uninstall
```

### Managing Application Paths

```bash
# Set path for portable Photoshop
launcher.exe -set-path photoshop=D:\PhotoshopPortable\PhotoshopCS6Portable.exe

# Set path for Acrobat
launcher.exe -set-path acrobat=D:\AcrobatReader\AcrobatDC.exe

# Set path for Microsoft Word
launcher.exe -set-path word=D:\MSOffice\WINWORD.EXE

# View all custom paths
launcher.exe -list-paths

# Remove a custom path
launcher.exe -remove-path photoshop
```

### Managing File Associations

```bash
# Associate PDF files with Photoshop
launcher.exe -set-assoc .pdf=photoshop

# Associate JPG files with Photoshop
launcher.exe -set-assoc .jpg=photoshop

# Associate DOCX files with Word
launcher.exe -set-assoc .docx=word

# View all file associations
launcher.exe -list-assoc

# Remove a file association
launcher.exe -remove-assoc .pdf
```

## Real-World Examples

### Example 1: Setting up Portable Photoshop

```bash
# 1. Set Photoshop path
launcher.exe -set-path photoshop=D:\PortableApps\PhotoshopCS6\PhotoshopCS6Portable.exe

# 2. Set up file associations
launcher.exe -set-assoc .jpg=photoshop
launcher.exe -set-assoc .jpeg=photoshop
launcher.exe -set-assoc .png=photoshop
launcher.exe -set-assoc .psd=photoshop
launcher.exe -set-assoc .pdf=photoshop

# 3. Verify settings
launcher.exe -list-paths
launcher.exe -list-assoc
```

### Example 2: Multiple Application Setup

```bash
# 1. Set application paths
launcher.exe -set-path photoshop=D:\PortableApps\PhotoshopCS6\PhotoshopCS6Portable.exe
launcher.exe -set-path acrobat=D:\PortableApps\AcrobatReader\AcrobatDC.exe
launcher.exe -set-path word=D:\PortableApps\MSOffice\WINWORD.EXE

# 2. Set file associations
launcher.exe -set-assoc .pdf=acrobat
launcher.exe -set-assoc .jpg=photoshop
launcher.exe -set-assoc .png=photoshop
launcher.exe -set-assoc .docx=word

# 3. View all settings
launcher.exe -list-paths
launcher.exe -list-assoc
```

### Example 3: Changing Existing Associations

```bash
# 1. Check current associations
launcher.exe -list-assoc

# 2. Change PDF from Acrobat to Photoshop
launcher.exe -set-assoc .pdf=photoshop

# 3. Verify change
launcher.exe -list-assoc
```

### Example 4: Complete Reset

```bash
# 1. Remove all custom paths
launcher.exe -remove-path photoshop
launcher.exe -remove-path acrobat
launcher.exe -remove-path word

# 2. Remove all associations
launcher.exe -remove-assoc .pdf
launcher.exe -remove-assoc .jpg
launcher.exe -remove-assoc .png
launcher.exe -remove-assoc .docx

# 3. Verify everything is cleared
launcher.exe -list-paths
launcher.exe -list-assoc

# 4. Uninstall handler
launcher.exe -uninstall
```

## Configuration File

The launcher stores settings in `~/.launcher_config.json`:

```json
{
    "customPaths": {
        "photoshop": "D:\\PortableApps\\PhotoshopCS6\\PhotoshopCS6Portable.exe",
        "acrobat": "D:\\PortableApps\\AcrobatReader\\AcrobatDC.exe",
        "word": "D:\\PortableApps\\MSOffice\\WINWORD.EXE"
    },
    "customFileAssociations": {
        ".pdf": "acrobat",
        ".jpg": "photoshop",
        ".png": "photoshop",
        ".docx": "word"
    }
}
```

## Logging

- Log files are stored in: `~/hrep_launcher_logs/`
- Format: `launcher_YYYY-MM-DD.log`
- Contains detailed operation logs for troubleshooting

## Troubleshooting

### Application Won't Open

1. Check if path exists:
```bash
launcher.exe -list-paths
```

2. Try removing and resetting the path:
```bash
launcher.exe -remove-path photoshop
launcher.exe -set-path photoshop=D:\correct\path\to\photoshop.exe
```

### Wrong Application Opens

1. Check current associations:
```bash
launcher.exe -list-assoc
```

2. Remove incorrect association and set correct one:
```bash
launcher.exe -remove-assoc .pdf
launcher.exe -set-assoc .pdf=acrobat
```

### Elevation Required

If you get "operation requires elevation" error:
1. Run command prompt as administrator
2. Reinstall the handler:
```bash
launcher.exe -uninstall
launcher.exe -install
```

## Supported File Types

Default supported extensions:
- `.jpg`, `.jpeg`, `.png`, `.psd` (Photoshop)
- `.pdf` (Acrobat)
- `.docx` (Word)

You can add custom associations for any extension using `-set-assoc`.