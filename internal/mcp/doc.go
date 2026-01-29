// Package mcp implements a Model Context Protocol (MCP) server for Spartan Scraper.
//
// Responsibilities:
// - Implement a JSON-RPC 2.0 based server over stdio.
// - Expose Spartan capabilities (scrape, crawl, research) as MCP tools.
// - Manage lifecycle and state for MCP sessions.
//
// Does NOT handle:
// - Implementation of the scraping or crawling logic itself.
// - Authentication/Authorization beyond what Spartan handles internally.
//
// Invariants/Assumptions:
// - Communicates over stdin/stdout as defined by the MCP stdio transport.
// - Expects a valid Spartan configuration and initialized services.
package mcp
