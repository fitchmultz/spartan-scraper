/**
 * Purpose: Bootstrap the Spartan Scraper web application and mount the root provider tree.
 * Responsibilities: Configure the generated API client, locate the React root, and render the app inside the shared StrictMode and toast-notification boundaries.
 * Scope: Client-side startup only.
 * Usage: Executed by Vite as the browser entrypoint.
 * Invariants/Assumptions: `#root` exists in `index.html`, API client configuration is side-effectful but idempotent, and the toast provider wraps the entire app.
 */

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import { App } from "./App";
import { ToastProvider } from "./components/toast";
import { configureApiClient } from "./lib/api-config";

const root = document.getElementById("root");
if (!root) {
  throw new Error("Missing #root element");
}

configureApiClient();

createRoot(root).render(
  <StrictMode>
    <ToastProvider>
      <App />
    </ToastProvider>
  </StrictMode>,
);
