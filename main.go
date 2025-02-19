package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/jchv/go-webview2"
)

func main() {
	logFile := setupLogging()
	if logFile != nil {
		defer logFile.Close()
	}

	log.Printf("Launcher started with args: %v", os.Args)

	// Handle URI directly if passed as argument
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "launcher://") {
		log.Printf("Processing direct URI: %s", os.Args[1])
		if err := processURI(os.Args[1]); err != nil {
			log.Printf("Error processing URI: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// CLI flags
	cliMode := flag.Bool("cli", false, "Run in CLI mode")
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

	// If any CLI flag is set, run in CLI mode
	if *cliMode || *installFlag || *uninstallFlag || *uriFlag != "" ||
		*setPathFlag != "" || *listPathsFlag || *removePathFlag != "" ||
		*setAssocFlag != "" || *listAssocFlag || *removeAssocFlag != "" {
		handleCLI(installFlag, uninstallFlag, uriFlag, setPathFlag, listPathsFlag,
			removePathFlag, setAssocFlag, listAssocFlag, removeAssocFlag)
	} else {
		// No CLI flags, run GUI
		startGUI()
	}
}

func handleCLI(installFlag, uninstallFlag *bool, uriFlag, setPathFlag *string,
	listPathsFlag *bool, removePathFlag, setAssocFlag *string,
	listAssocFlag *bool, removeAssocFlag *string) {
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

func startGUI() {
	// Create webview with options
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     true,
		AutoFocus: true,
	})
	if w == nil {
		log.Fatal("Failed to create webview")
	}
	defer w.Destroy()
	w.SetTitle("LIRMDS Client Launcher Configuration")
	w.SetSize(800, 600, webview2.HintNone)

	// Bind Go functions to JavaScript
	w.Bind("getConfig", func() string {
		config, err := loadConfig()
		if err != nil {
			config = Config{
				CustomPaths:            make(map[string]string),
				CustomFileAssociations: make(map[string]string),
			}
		}
		data, _ := json.Marshal(config)
		return string(data)
	})

	w.Bind("saveConfig", func(jsonStr string) string {
		var config Config
		if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		if err := saveConfig(config); err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return "Success"
	})

	w.Bind("installHandler", func() string {
		if err := installHandler(); err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return "Success"
	})

	w.Bind("uninstallHandler", func() string {
		if err := uninstallHandler(); err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return "Success"
	})

	w.Bind("selectFile", func() string {
		// Create a new dialog
		cmd := exec.Command("powershell", "-Command", `
            Add-Type -AssemblyName System.Windows.Forms
            $dialog = New-Object System.Windows.Forms.OpenFileDialog
            $dialog.Filter = "Executable files (*.exe)|*.exe|All files (*.*)|*.*"
            if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
                $dialog.FileName
            }
        `)

		output, err := cmd.Output()
		if err != nil {
			return ""
		}

		// Trim any whitespace or newlines and remove any "OK" prefix
		path := strings.TrimSpace(string(output))
		path = strings.TrimPrefix(path, "OK")
		return path
	})

	// HTML content for the GUI
	htmlContent := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>LIRMDS Client Launcher Configuration</title>
    <style>
        body { 
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        .section {
            margin-bottom: 20px;
            padding: 15px;
            border: 1px solid #ddd;
            border-radius: 4px;
        }
        button {
            background: #0066cc;
            color: white;
            border: none;
            padding: 8px 16px;
            border-radius: 4px;
            cursor: pointer;
            margin: 4px;
            white-space: nowrap;
        }
        button:hover {
            background: #0052a3;
        }
        .entry {
            display: flex;
            margin: 8px 0;
            align-items: center;
            gap: 8px;
        }
        input, select {
            padding: 6px;
            border: 1px solid #ddd;
            border-radius: 4px;
        }
        input[type="text"] {
            flex: 1;
        }
        .remove-button {
            background: #dc3545;
        }
        #appPath {
            background-color: #f8f8f8;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>LIRMDS Client Launcher Configuration</h1>
        <h2>Application Paths</h2>
        <div class="section">
            <div id="pathsList"></div>
            <div class="entry">
                <input type="text" id="appName" placeholder="Application name">
                <input type="text" id="appPath" placeholder="Application path" readonly>
                <button onclick="browseFile()">Browse</button>
                <button onclick="addPath()">Add Path</button>
            </div>
        </div>

        <h2>File Associations</h2>
        <div class="section">
            <div id="assocList"></div>
            <div class="entry">
                <input type="text" id="fileExt" placeholder="File extension (e.g. .pdf)">
                <select id="appSelect"></select>
                <button onclick="addAssociation()">Add Association</button>
            </div>
        </div>

        <h2>Installation</h2>
        <div class="section">
            <div class="entry">
                <button onclick="install()">Install Protocol Handler</button>
                <button onclick="uninstall()" class="remove-button">Uninstall Protocol Handler</button>
            </div>
        </div>
    </div>

    <script>
        let config = null;

        async function loadConfig() {
            config = JSON.parse(await getConfig());
            updateUI();
        }

        async function browseFile() {
            const filePath = await selectFile();
            if (filePath) {
                document.getElementById('appPath').value = filePath;
            }
        }

        function updateUI() {
            const pathsList = document.getElementById('pathsList');
            pathsList.innerHTML = '';
            for (const [app, path] of Object.entries(config.customPaths)) {
                const div = document.createElement('div');
                div.className = 'entry';
                div.innerHTML = 
                    '<span style="flex: 1;">' + app + ': ' + path + '</span>' +
                    '<button onclick="removePath(\'' + app + '\')" class="remove-button">Remove</button>';
                pathsList.appendChild(div);
            }

            const assocList = document.getElementById('assocList');
            assocList.innerHTML = '';
            for (const [ext, app] of Object.entries(config.customFileAssociations)) {
                const div = document.createElement('div');
                div.className = 'entry';
                div.innerHTML = 
                    '<span style="flex: 1;">' + ext + ' - ' + app + '</span>' +
                    '<button onclick="removeAssociation(\'' + ext + '\')" class="remove-button">Remove</button>';
                assocList.appendChild(div);
            }

            const appSelect = document.getElementById('appSelect');
            appSelect.innerHTML = '';
            for (const app of Object.keys(config.customPaths)) {
                const option = document.createElement('option');
                option.value = app;
                option.textContent = app;
                appSelect.appendChild(option);
            }
        }

        async function addPath() {
            const appName = document.getElementById('appName').value;
            const appPath = document.getElementById('appPath').value;
            if (!appName || !appPath) {
                alert('Please enter both application name and path');
                return;
            }

            config.customPaths[appName] = appPath;
            const result = await saveConfig(JSON.stringify(config));
            if (result === 'Success') {
                document.getElementById('appName').value = '';
                document.getElementById('appPath').value = '';
                updateUI();
            } else {
                alert('Error saving config: ' + result);
            }
        }

        async function removePath(app) {
            if (confirm('Are you sure you want to remove this path?')) {
                delete config.customPaths[app];
                const result = await saveConfig(JSON.stringify(config));
                if (result === 'Success') {
                    updateUI();
                } else {
                    alert('Error saving config: ' + result);
                }
            }
        }

        async function addAssociation() {
            const fileExt = document.getElementById('fileExt').value;
            const appSelect = document.getElementById('appSelect');
            const app = appSelect.value;
            if (!fileExt || !app) {
                alert('Please enter both file extension and select an application');
                return;
            }

            let ext = fileExt;
            if (!ext.startsWith('.')) {
                ext = '.' + ext;
            }

            config.customFileAssociations[ext] = app;
            const result = await saveConfig(JSON.stringify(config));
            if (result === 'Success') {
                document.getElementById('fileExt').value = '';
                updateUI();
            } else {
                alert('Error saving config: ' + result);
            }
        }

        async function removeAssociation(ext) {
            if (confirm('Are you sure you want to remove this association?')) {
                delete config.customFileAssociations[ext];
                const result = await saveConfig(JSON.stringify(config));
                if (result === 'Success') {
                    updateUI();
                } else {
                    alert('Error saving config: ' + result);
                }
            }
        }

        async function install() {
            const result = await installHandler();
            alert(result === 'Success' ? 'Installation successful' : 'Installation failed: ' + result);
        }

        async function uninstall() {
            const result = await uninstallHandler();
            alert(result === 'Success' ? 'Uninstallation successful' : 'Uninstallation failed: ' + result);
        }

        loadConfig();
    </script>
</body>
</html>`

	// Set the HTML content and run the webview
	w.Navigate("data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(htmlContent)))
	w.Run()
}
