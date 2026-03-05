# Spartan Scraper Browser Extension

A browser extension for quick scraping with Spartan - right from your browser.

## Features

- **Quick Scrape**: Click the extension icon to scrape the current page
- **Context Menu**: Right-click any page or link to scrape with Spartan
- **Template Selection**: Choose from available extraction templates
- **Job Status**: Monitor scrape job status in real-time
- **API Key Configuration**: Securely store your Spartan API key

## Installation

### From Source (Developer Mode)

1. **Build the extension**:
   ```bash
   cd extensions
   pnpm install
   pnpm run build
   ```

2. **Open Chrome/Edge and navigate to**:
   ```
   chrome://extensions
   ```

3. **Enable Developer Mode** (toggle in top-right corner)

4. **Click "Load unpacked"** and select the `extensions/dist` folder

5. **Configure the extension**:
   - Click the extension icon
   - Click the settings (gear) icon
   - Enter your Spartan server URL (default: `http://localhost:8741`)
   - Enter your API key (generate with `spartan auth apikey generate`)
   - Click "Save Settings"

### Firefox Installation

1. **Build the extension**:
   ```bash
   cd extensions
   pnpm install
   pnpm run build
   ```

2. **Open Firefox and navigate to**:
   ```
   about:debugging
   ```

3. **Click "This Firefox"** → **"Load Temporary Add-on"**

4. **Select the `manifest.json` file** from the `extensions/dist` folder

## Usage

### Quick Scrape Current Page

1. Navigate to any webpage you want to scrape
2. Click the Spartan extension icon in your browser toolbar
3. Select a template (optional)
4. Click "Scrape Page"
5. Wait for the job to complete
6. Click "View Results" to see the extracted data

### Context Menu Scrape

1. Right-click on any webpage
2. Select "Scrape with Spartan" from the context menu
3. The popup will open with the URL pre-filled
4. Configure options and click "Scrape Page"

### Scrape a Link

1. Right-click on any link
2. Select "Scrape with Spartan"
3. The popup will open with the link URL pre-filled

## Configuration

### API Key Setup

Before using the extension, you need to configure your API key:

1. **Generate an API key** using the Spartan CLI:
   ```bash
   spartan auth apikey generate
   ```

2. **Open extension options**:
   - Click the extension icon
   - Click the settings (gear) icon
   - Or right-click the extension icon and select "Options"

3. **Enter your settings**:
   - **Server URL**: Your Spartan API server URL (default: `http://localhost:8741`)
   - **API Key**: The key generated from the CLI
   - **Default Template**: Template to pre-select (optional)
   - **Headless Mode**: Whether to use headless browser by default

4. **Click "Save Settings"**

5. **Test the connection** by clicking "Test Connection"

## Development

### Project Structure

```
extensions/
├── manifest.json          # Extension manifest (Manifest V3)
├── package.json           # Dependencies and scripts
├── tsconfig.json          # TypeScript configuration
├── icons/                 # Extension icons
│   ├── icon.svg
│   ├── icon16.png
│   ├── icon32.png
│   ├── icon48.png
│   └── icon128.png
├── shared/                # Shared code
│   ├── types.ts           # TypeScript type definitions
│   ├── storage.ts         # Chrome storage helpers
│   └── api.ts             # API client
├── background/            # Service worker
│   └── background.ts      # Background script
├── popup/                 # Extension popup
│   ├── popup.html
│   ├── popup.css
│   └── popup.ts
├── options/               # Options page
│   ├── options.html
│   ├── options.css
│   └── options.ts
└── content/               # Content script
    └── content.ts
```

### Build Scripts

```bash
# Install dependencies
pnpm install

# Build extension
pnpm run build

# Watch mode for development
pnpm run dev

# Clean build artifacts
pnpm run clean

# Package for distribution
pnpm run package
```

### API Integration

The extension communicates with the Spartan API using these endpoints:

- `GET /v1/templates` - List available extraction templates
- `POST /v1/scrape` - Create a new scrape job
- `GET /v1/jobs/{id}` - Get job status

Authentication is done via the `X-API-Key` header.

## Troubleshooting

### "API key not configured" error

- Open extension options and configure your API key
- Generate an API key with `spartan auth apikey generate`

### "Cannot connect to server" error

- Ensure your Spartan server is running (`spartan server`)
- Check that the Server URL in options is correct
- Verify the server is accessible from your browser

### Extension not working

- Check the browser console for errors
- Verify the API key is correct
- Try reloading the extension from `chrome://extensions`

## Browser Support

- Chrome (Manifest V3)
- Edge (Manifest V3)
- Firefox (Manifest V3 with polyfill)

## License

Same as the main Spartan Scraper project.
