/**
 * Spartan Scraper Extension - Content Script
 *
 * Injected into web pages to detect page context and enable
 * element picking for visual selector (future enhancement).
 */

import type { TabInfo } from "../shared/types.js";

/**
 * Extract page metadata for scraping context
 */
function getPageMetadata(): TabInfo {
  const url = window.location.href;
  const title = document.title || "";

  // Try to get favicon
  let favicon: string | undefined;
  const faviconLink =
    document.querySelector('link[rel="icon"]') ||
    document.querySelector('link[rel="shortcut icon"]');
  if (faviconLink) {
    const href = faviconLink.getAttribute("href");
    if (href) {
      favicon = new URL(href, url).href;
    }
  }

  return { url, title, favicon };
}

/**
 * Extract OpenGraph metadata
 */
function getOpenGraphMetadata(): Record<string, string> {
  const meta: Record<string, string> = {};

  const ogTags = document.querySelectorAll('meta[property^="og:"]');
  ogTags.forEach((tag) => {
    const property = tag.getAttribute("property");
    const content = tag.getAttribute("content");
    if (property && content) {
      meta[property] = content;
    }
  });

  return meta;
}

/**
 * Extract article/content metadata
 */
function getArticleMetadata(): Record<string, string> {
  const meta: Record<string, string> = {};

  // Article published/modified dates
  const articlePublished = document.querySelector(
    'meta[property="article:published_time"], meta[name="publishedDate"]',
  );
  if (articlePublished) {
    meta.publishedTime = articlePublished.getAttribute("content") || "";
  }

  // Author
  const author = document.querySelector(
    'meta[name="author"], meta[property="article:author"]',
  );
  if (author) {
    meta.author = author.getAttribute("content") || "";
  }

  // Description
  const description = document.querySelector(
    'meta[name="description"], meta[property="og:description"]',
  );
  if (description) {
    meta.description = description.getAttribute("content") || "";
  }

  return meta;
}

/**
 * Check if the page is likely an article/content page
 */
function isArticlePage(): boolean {
  // Check for article HTML5 tag
  if (document.querySelector("article")) {
    return true;
  }

  // Check for article schema.org type
  const schemas = document.querySelectorAll('[itemtype*="Article"]');
  if (schemas.length > 0) {
    return true;
  }

  // Check for OpenGraph article type
  const ogType = document.querySelector('meta[property="og:type"]');
  if (ogType?.getAttribute("content")?.includes("article")) {
    return true;
  }

  return false;
}

/**
 * Get suggested template based on page type
 */
function getSuggestedTemplate(): string {
  if (isArticlePage()) {
    return "article";
  }

  // Check for product pages
  if (
    document.querySelector('[itemtype*="Product"]') ||
    document.querySelector(".product") ||
    document.querySelector("#product")
  ) {
    return "product";
  }

  return "";
}

/**
 * Handle messages from the background script
 */
chrome.runtime.onMessage.addListener((request, _sender, sendResponse) => {
  if (request.type === "GET_PAGE_CONTEXT") {
    const context = {
      metadata: getPageMetadata(),
      openGraph: getOpenGraphMetadata(),
      article: getArticleMetadata(),
      suggestedTemplate: getSuggestedTemplate(),
    };
    sendResponse({ success: true, data: context });
    return true;
  }

  return false;
});

// Log initialization (only in development builds)
// @ts-expect-error - __DEV__ is injected by build process// eslint-disable-next-line
if (typeof __DEV__ !== "undefined" && __DEV__) {
  console.log("Spartan Scraper: Content script initialized");
}
