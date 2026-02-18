# Azure Web Apps – Playwright startup

Use the startup script so Playwright and its browsers are installed when the app starts.

## Option 1: Startup command (Linux App Service)

1. In **Azure Portal** → your Web App → **Configuration** → **General settings**.
2. Set **Startup Command** to:
   ```bash
   bash -c "curl -sL https://raw.githubusercontent.com/YOUR_ORG/YOUR_REPO/main/azure/startup-playwright.sh | bash"
   ```
   Or, if the script is in your deployment (e.g. from GitHub Actions / ZIP deploy), point to the file in your app root:
   ```bash
   /home/site/wwwroot/azure/startup-playwright.sh
   ```
   Ensure the file is executable (`chmod +x azure/startup-playwright.sh` before deploy).

## Option 2: Run script from repo (recommended)

Deploy your app so `azure/startup-playwright.sh` is at `/home/site/wwwroot/azure/startup-playwright.sh`, then set **Startup Command** to:

```bash
bash /home/site/wwwroot/azure/startup-playwright.sh
```

## Customization

- **App path**  
  Default is `/home/site/wwwroot`. Override with:
  ```bash
  STARTUP_APP_PATH=/home/site/wwwroot bash /home/site/wwwroot/azure/startup-playwright.sh
  ```

- **Entrypoint**  
  By default the script runs `node server.js`, or `npm start` if there is no `server.js`. To override, set the **Application setting** `STARTUP_CMD` in Azure (e.g. `npm start` or `node dist/index.js`). You can also edit the last lines in `startup-playwright.sh`.

- **Browsers**  
  The script installs only **Chromium** to keep size down. To add Firefox/WebKit, change the install line to:
  ```bash
  npx playwright install chromium firefox webkit
  ```

## Option 3: Custom Docker image

For faster cold starts and no install at runtime, build an image that already has Playwright and browsers, and use that image as the Web App’s container. Use this script’s install steps in your Dockerfile instead of at startup.
