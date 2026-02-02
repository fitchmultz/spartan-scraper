/**
 * Spartan Scraper Extension - Options Page Script
 *
 * Handles the options/settings page for configuring the extension.
 */

import { getSettings, saveSettings } from "../shared/storage.js";
import { testConnection, getTemplates } from "../shared/api.js";
import type { ExtensionSettings } from "../shared/types.js";

// DOM Elements
const settingsForm = document.getElementById("settingsForm") as HTMLFormElement;
const apiUrlInput = document.getElementById("apiUrl") as HTMLInputElement;
const apiKeyInput = document.getElementById("apiKey") as HTMLInputElement;
const toggleKeyBtn = document.getElementById("toggleKeyBtn") as HTMLButtonElement;
const eyeIcon = document.getElementById("eyeIcon") as HTMLElement;
const eyeOffIcon = document.getElementById("eyeOffIcon") as HTMLElement;
const testConnectionBtn = document.getElementById(
  "testConnectionBtn",
) as HTMLButtonElement;
const testResult = document.getElementById("testResult") as HTMLDivElement;
const testResultIcon = document.getElementById("testResultIcon") as HTMLSpanElement;
const testResultText = document.getElementById("testResultText") as HTMLSpanElement;
const defaultTemplateSelect = document.getElementById(
  "defaultTemplate",
) as HTMLSelectElement;
const defaultHeadlessCheck = document.getElementById(
  "defaultHeadless",
) as HTMLInputElement;
const toast = document.getElementById("toast") as HTMLDivElement;
const toastIcon = document.getElementById("toastIcon") as HTMLSpanElement;
const toastMessage = document.getElementById("toastMessage") as HTMLSpanElement;

/**
 * Initialize the options page
 */
async function init(): Promise<void> {
  try {
    // Load saved settings
    const settings = await getSettings();
    apiUrlInput.value = settings.apiUrl;
    apiKeyInput.value = settings.apiKey;
    defaultHeadlessCheck.checked = settings.defaultHeadless;

    // Load templates for default template dropdown
    await loadTemplates(settings);

    // Set default template if configured
    if (settings.defaultTemplate) {
      defaultTemplateSelect.value = settings.defaultTemplate;
    }
  } catch (err) {
    showToast("Failed to load settings", "error");
    console.error("Failed to load settings:", err);
  }
}

/**
 * Load available templates
 */
async function loadTemplates(settings: ExtensionSettings): Promise<void> {
  if (!settings.apiKey) {
    return;
  }

  try {
    const templates = await getTemplates(settings.apiUrl, settings.apiKey);

    // Clear existing options except the first
    while (defaultTemplateSelect.options.length > 1) {
      defaultTemplateSelect.remove(1);
    }

    // Add templates
    templates.forEach((template) => {
      const option = document.createElement("option");
      option.value = template;
      option.textContent = template;
      defaultTemplateSelect.appendChild(option);
    });
  } catch (err) {
    console.error("Failed to load templates:", err);
  }
}

/**
 * Handle form submission
 */
async function handleSubmit(event: Event): Promise<void> {
  event.preventDefault();

  const settings: ExtensionSettings = {
    apiUrl: apiUrlInput.value.trim() || "http://localhost:8741",
    apiKey: apiKeyInput.value.trim(),
    defaultTemplate: defaultTemplateSelect.value,
    defaultHeadless: defaultHeadlessCheck.checked,
  };

  try {
    await saveSettings(settings);
    showToast("Settings saved successfully", "success");
  } catch (err) {
    showToast("Failed to save settings", "error");
    console.error("Failed to save settings:", err);
  }
}

/**
 * Toggle API key visibility
 */
function toggleKeyVisibility(): void {
  const isPassword = apiKeyInput.type === "password";
  apiKeyInput.type = isPassword ? "text" : "password";
  eyeIcon.classList.toggle("hidden", !isPassword);
  eyeOffIcon.classList.toggle("hidden", isPassword);
}

/**
 * Test API connection
 */
async function handleTestConnection(): Promise<void> {
  const apiUrl = apiUrlInput.value.trim() || "http://localhost:8741";
  const apiKey = apiKeyInput.value.trim();

  if (!apiKey) {
    showTestResult(false, "Please enter an API key");
    return;
  }

  testConnectionBtn.disabled = true;
  testConnectionBtn.textContent = "Testing...";

  try {
    const result = await testConnection(apiUrl, apiKey);
    showTestResult(result.success, result.message);
  } catch (err) {
    showTestResult(false, err instanceof Error ? err.message : "Connection failed");
  } finally {
    testConnectionBtn.disabled = false;
    testConnectionBtn.textContent = "Test Connection";
  }
}

/**
 * Show test result
 */
function showTestResult(success: boolean, message: string): void {
  testResult.classList.remove("hidden", "success", "error");
  testResult.classList.add(success ? "success" : "error");
  testResultIcon.textContent = success ? "✓" : "✗";
  testResultText.textContent = message;
}

/**
 * Show toast notification
 */
function showToast(message: string, type: "success" | "error"): void {
  toast.classList.remove("hidden", "success", "error");
  toast.classList.add(type);
  toastIcon.textContent = type === "success" ? "✓" : "✗";
  toastMessage.textContent = message;

  // Hide after 3 seconds
  setTimeout(() => {
    toast.classList.add("hidden");
  }, 3000);
}

// Event listeners
document.addEventListener("DOMContentLoaded", init);
settingsForm.addEventListener("submit", handleSubmit);
toggleKeyBtn.addEventListener("click", toggleKeyVisibility);
testConnectionBtn.addEventListener("click", handleTestConnection);
