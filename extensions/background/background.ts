/**
 * Spartan Scraper Extension - Background Service Worker
 *
 * Handles API calls, context menu, and message passing between
 * popup/content scripts and the Spartan API.
 */

import {
  createScrapeJob,
  getJobStatus,
  getTemplates,
  testConnection,
} from "../shared/api.js";
import { getSettings, getCachedTemplates, cacheTemplates } from "../shared/storage.js";
import type { Message, TabInfo } from "../shared/types.js";

// Context menu ID
const CONTEXT_MENU_ID = "spartan-scrape";

/**
 * Initialize extension on install/update
 */
chrome.runtime.onInstalled.addListener((details) => {
  console.log("Spartan Scraper extension installed:", details.reason);

  // Create context menu item
  chrome.contextMenus.create({
    id: CONTEXT_MENU_ID,
    title: "Scrape with Spartan",
    contexts: ["page", "link"],
    documentUrlPatterns: ["http://*/*", "https://*/*"],
  });
});

/**
 * Handle context menu clicks
 */
chrome.contextMenus.onClicked.addListener(async (info, tab) => {
  if (info.menuItemId !== CONTEXT_MENU_ID) return;

  const url = info.linkUrl || info.pageUrl;
  if (!url || !tab?.id) return;

  // Store the URL to scrape and open the popup
  await chrome.storage.session.set({ contextMenuUrl: url });

  // Open the popup action
  chrome.action.openPopup();
});

/**
 * Handle messages from popup and content scripts
 */
chrome.runtime.onMessage.addListener((message: Message, sender, sendResponse) => {
  // Return true to indicate async response
  const handleAsync = async (): Promise<void> => {
    try {
      switch (message.type) {
        case "GET_CURRENT_TAB": {
          const tabInfo = await getCurrentTabInfo();
          sendResponse({ success: true, data: tabInfo });
          break;
        }

        case "GET_TEMPLATES": {
          const templates = await fetchTemplates();
          sendResponse({ success: true, data: templates });
          break;
        }

        case "CREATE_SCRAPE_JOB": {
          const { url, template, headless } = message.payload as {
            url: string;
            template: string;
            headless: boolean;
          };
          const job = await submitScrapeJob(url, template, headless);
          sendResponse({ success: true, data: job });
          break;
        }

        case "GET_JOB_STATUS": {
          const { jobId } = message.payload as { jobId: string };
          const job = await fetchJobStatus(jobId);
          sendResponse({ success: true, data: job });
          break;
        }

        case "OPEN_OPTIONS_PAGE": {
          chrome.runtime.openOptionsPage();
          sendResponse({ success: true });
          break;
        }

        default:
          sendResponse({ success: false, error: "Unknown message type" });
      }
    } catch (err) {
      console.error("Background script error:", err);
      sendResponse({
        success: false,
        error: err instanceof Error ? err.message : "Unknown error",
      });
    }
  };

  handleAsync();
  return true;
});

/**
 * Get information about the currently active tab
 */
async function getCurrentTabInfo(): Promise<TabInfo> {
  const tabs = await chrome.tabs.query({ active: true, currentWindow: true });

  if (!tabs.length || !tabs[0].url) {
    throw new Error("No active tab found");
  }

  const tab = tabs[0];

  // Check if there's a context menu URL stored
  const session = await chrome.storage.session.get("contextMenuUrl");
  const url = session.contextMenuUrl || tab.url;

  // Clear the context menu URL
  if (session.contextMenuUrl) {
    await chrome.storage.session.remove("contextMenuUrl");
  }

  return {
    url,
    title: tab.title || "",
    favicon: tab.favIconUrl,
  };
}

/**
 * Fetch templates list with caching
 */
async function fetchTemplates(): Promise<string[]> {
  // Check cache first
  const cached = await getCachedTemplates();
  if (cached) {
    return cached;
  }

  const settings = await getSettings();

  if (!settings.apiKey) {
    throw new Error("API key not configured. Please set it in extension options.");
  }

  const templates = await getTemplates(settings.apiUrl, settings.apiKey);

  // Cache the result
  await cacheTemplates(templates);

  return templates;
}

/**
 * Submit a scrape job to the API
 */
async function submitScrapeJob(
  url: string,
  template: string,
  headless: boolean,
): Promise<unknown> {
  const settings = await getSettings();

  if (!settings.apiKey) {
    throw new Error("API key not configured. Please set it in extension options.");
  }

  const request = {
    url,
    headless,
    extract: template ? { template } : undefined,
    timeoutSeconds: 30,
  };

  return createScrapeJob(settings.apiUrl, settings.apiKey, request);
}

/**
 * Fetch job status from the API
 */
async function fetchJobStatus(jobId: string): Promise<unknown> {
  const settings = await getSettings();

  if (!settings.apiKey) {
    throw new Error("API key not configured");
  }

  return getJobStatus(settings.apiUrl, settings.apiKey, jobId);
}

/**
 * Test API connection (used by options page)
 */
export async function checkConnection(
  baseUrl: string,
  apiKey: string,
): Promise<{ success: boolean; message: string }> {
  return testConnection(baseUrl, apiKey);
}
