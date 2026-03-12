/**
 * React Application Entry Point
 *
 * This file initializes the Spartan Scraper web application by:
 * 1. Locating the root DOM element (#root)
 * 2. Creating a React root with concurrent rendering
 * 3. Rendering the main App component in StrictMode
 *
 * @module main
 */
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import { App } from "./App";
import { configureApiClient } from "./lib/api-config";

const root = document.getElementById("root");
if (!root) {
  throw new Error("Missing #root element");
}

configureApiClient();

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
