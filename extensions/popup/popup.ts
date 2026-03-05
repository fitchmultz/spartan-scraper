/**
 * Spartan Scraper Extension - Popup Script
 *
 * Handles the popup UI interactions and communicates with the
 * background script to perform scraping operations.
 */

import type { Job, JobStatus, TabInfo } from "../shared/types.js";

// DOM Elements
const urlInput = document.getElementById("urlInput") as HTMLInputElement;
const templateSelect = document.getElementById(
  "templateSelect",
) as HTMLSelectElement;
const headlessCheck = document.getElementById(
  "headlessCheck",
) as HTMLInputElement;
const scrapeBtn = document.getElementById("scrapeBtn") as HTMLButtonElement;
const scrapeBtnText = document.getElementById(
  "scrapeBtnText",
) as HTMLSpanElement;
const scrapeBtnLoader = document.getElementById(
  "scrapeBtnLoader",
) as HTMLSpanElement;
const settingsBtn = document.getElementById(
  "settingsBtn",
) as HTMLButtonElement;
const configureBtn = document.getElementById(
  "configureBtn",
) as HTMLButtonElement;
const configWarning = document.getElementById(
  "configWarning",
) as HTMLDivElement;
const statusSection = document.getElementById(
  "statusSection",
) as HTMLDivElement;
const statusBadge = document.getElementById(
  "statusBadge",
) as HTMLSpanElement;
const statusMessage = document.getElementById(
  "statusMessage",
) as HTMLDivElement;
const statusActions = document.getElementById(
  "statusActions",
) as HTMLDivElement;
const viewResultsLink = document.getElementById(
  "viewResultsLink",
) as HTMLAnchorElement;
const errorDisplay = document.getElementById(
  "errorDisplay",
) as HTMLDivElement;
const errorText = document.getElementById("errorText") as HTMLSpanElement;

// State
let currentJobId: string | null = null;
let pollInterval: number | null = null;

/**
 * Initialize the popup
 */
async function init(): Promise<void> {
  try {
    // Load current tab info
    const tabInfo = await sendMessage<TabInfo>("GET_CURRENT_TAB");
    if (tabInfo) {
      urlInput.value = tabInfo.url;
    }

    // Load templates
    await loadTemplates();

    // Check if API key is configured
    await checkConfiguration();
  } catch (err) {
    showError(err instanceof Error ? err.message : "Failed to initialize");
  }
}

/**
 * Load templates from the API
 */
async function loadTemplates(): Promise<void> {
  try {
    const templates = await sendMessage<string[]>("GET_TEMPLATES");

    // Clear loading option
    templateSelect.innerHTML = "";

    // Add default option
    const defaultOption = document.createElement("option");
    defaultOption.value = "";
    defaultOption.textContent = "Default (no template)";
    templateSelect.appendChild(defaultOption);

    // Add templates
    if (templates && templates.length > 0) {
      templates.forEach((template) => {
        const option = document.createElement("option");
        option.value = template;
        option.textContent = template;
        templateSelect.appendChild(option);
      });
    }
  } catch (err) {
    // If templates fail to load, keep the default option
    templateSelect.innerHTML =
      '<option value="">Default (no template)</option>';
    console.error("Failed to load templates:", err);
  }
}

/**
 * Check if API key is configured
 */
async function checkConfiguration(): Promise<void> {
  try {
    // Try to fetch templates - this will fail if API key is missing
    await sendMessage<string[]>("GET_TEMPLATES");
    configWarning.classList.add("hidden");
  } catch (err) {
    const message = err instanceof Error ? err.message : "";
    if (
      message.includes("API key") ||
      message.includes("401") ||
      message.includes("Unauthorized")
    ) {
      configWarning.classList.remove("hidden");
    }
  }
}

/**
 * Handle scrape button click
 */
async function handleScrape(): Promise<void> {
  if (currentJobId) {
    // Reset if there's a current job
    resetJobState();
    return;
  }

  const url = urlInput.value.trim();
  if (!url) {
    showError("No URL to scrape");
    return;
  }

  const template = templateSelect.value;
  const headless = headlessCheck.checked;

  setLoading(true);
  hideError();
  statusSection.classList.remove("hidden");
  updateStatus("queued", "Submitting scrape job...");

  try {
    const job = await sendMessage<Job>("CREATE_SCRAPE_JOB", {
      url,
      template,
      headless,
    });

    currentJobId = job.id;
    updateStatus(job.status, `Job ${job.id} created`);

    // Start polling for status
    startPolling(job.id);
  } catch (err) {
    setLoading(false);
    updateStatus("failed", "Failed to create job");
    showError(err instanceof Error ? err.message : "Failed to create scrape job");
  }
}

/**
 * Poll for job status updates
 */
function startPolling(jobId: string): void {
  if (pollInterval) {
    clearInterval(pollInterval);
  }

  pollInterval = window.setInterval(async () => {
    try {
      const job = await sendMessage<Job>("GET_JOB_STATUS", { jobId });
      updateStatus(job.status, getStatusMessage(job));

      if (job.status === "succeeded" || job.status === "failed" || job.status === "canceled") {
        stopPolling();
        setLoading(false);

        if (job.status === "succeeded") {
          scrapeBtnText.textContent = "Scrape Another";
          statusActions.classList.remove("hidden");

          // Set up view results link
          const settings = await chrome.storage.sync.get("spartan_settings");
          const apiUrl = settings.spartan_settings?.apiUrl || "http://localhost:8741";
          viewResultsLink.href = `${apiUrl}/web/jobs/${jobId}`;
        } else if (job.status === "failed") {
          scrapeBtnText.textContent = "Try Again";
          showError(job.error || "Job failed");
        }
      }
    } catch (err) {
      console.error("Failed to poll job status:", err);
    }
  }, 2000);
}

/**
 * Stop polling for job status
 */
function stopPolling(): void {
  if (pollInterval) {
    clearInterval(pollInterval);
    pollInterval = null;
  }
}

/**
 * Reset job state for a new scrape
 */
function resetJobState(): void {
  currentJobId = null;
  stopPolling();
  setLoading(false);
  statusSection.classList.add("hidden");
  statusActions.classList.add("hidden");
  scrapeBtnText.textContent = "Scrape Page";
  hideError();
}

/**
 * Update the status display
 */
function updateStatus(status: JobStatus, message: string): void {
  statusBadge.textContent = status;
  statusBadge.className = `status-badge ${status}`;
  statusMessage.textContent = message;
}

/**
 * Get a human-readable status message
 */
function getStatusMessage(job: Job): string {
  switch (job.status) {
    case "queued":
      return "Waiting in queue...";
    case "running":
      return "Scraping in progress...";
    case "succeeded":
      return "Scrape completed successfully!";
    case "failed":
      return job.error || "Scrape failed";
    case "canceled":
      return "Job was canceled";
    default:
      return `Status: ${job.status}`;
  }
}

/**
 * Set loading state on the scrape button
 */
function setLoading(loading: boolean): void {
  scrapeBtn.disabled = loading;
  if (loading) {
    scrapeBtnText.classList.add("hidden");
    scrapeBtnLoader.classList.remove("hidden");
  } else {
    scrapeBtnText.classList.remove("hidden");
    scrapeBtnLoader.classList.add("hidden");
  }
}

/**
 * Show an error message
 */
function showError(message: string): void {
  errorText.textContent = message;
  errorDisplay.classList.remove("hidden");
}

/**
 * Hide the error message
 */
function hideError(): void {
  errorDisplay.classList.add("hidden");
}

/**
 * Open the options page
 */
function openOptions(): void {
  sendMessage("OPEN_OPTIONS_PAGE");
}

/**
 * Send a message to the background script
 */
function sendMessage<T>(type: string, payload?: unknown): Promise<T> {
  return new Promise((resolve, reject) => {
    chrome.runtime.sendMessage({ type, payload }, (response) => {
      if (chrome.runtime.lastError) {
        reject(new Error(chrome.runtime.lastError.message));
        return;
      }

      if (!response) {
        reject(new Error("No response from background script"));
        return;
      }

      if (response.success) {
        resolve(response.data as T);
      } else {
        reject(new Error(response.error || "Unknown error"));
      }
    });
  });
}

// Event listeners
document.addEventListener("DOMContentLoaded", init);
scrapeBtn.addEventListener("click", handleScrape);
settingsBtn.addEventListener("click", openOptions);
configureBtn.addEventListener("click", openOptions);
