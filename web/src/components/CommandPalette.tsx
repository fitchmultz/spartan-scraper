/**
 * Command Palette Component
 *
 * Provides a fuzzy-searchable command palette UI using the cmdk library.
 * Allows quick navigation between views, form submissions, and job management.
 * Groups commands by category: Actions, Navigation, and Jobs.
 *
 * @module CommandPalette
 */

import { useState, useEffect, useCallback, useMemo } from "react";
import { Command } from "cmdk";
import type { JobEntry } from "../types";
import type { JobPreset } from "../types/presets";

export type CommandPaletteProps = {
  /** Whether the palette is visible */
  isOpen: boolean;
  /** Callback when palette should close */
  onClose: () => void;
  /** List of jobs to display */
  jobs: JobEntry[];
  /** Navigate to a specific view */
  onNavigate: (view: "jobs" | "results" | "forms") => void;
  /** Submit a form by type */
  onSubmitForm: (formType: "scrape" | "crawl" | "research") => void;
  /** Cancel a job by ID */
  onCancelJob: (jobId: string) => void;

  /** Currently active/running job ID */
  activeJobId?: string;
  /** Platform indicator for shortcut display */
  isMac?: boolean;
  /** Available presets to select from */
  presets?: JobPreset[];
  /** Callback when a preset is selected */
  onSelectPreset?: (preset: JobPreset) => void;
  /** Callback to restart the onboarding tour */
  onRestartTour?: () => void;
};

type CommandItem = {
  id: string;
  label: string;
  shortcut?: string;
  icon?: string;
  group: "actions" | "navigation" | "jobs" | "presets";
  disabled?: boolean;
  onSelect: () => void;
};

/**
 * Format job status for display with icon.
 */
function getStatusIcon(status: JobEntry["status"]): string {
  switch (status) {
    case "running":
      return "▶️";
    case "succeeded":
      return "✅";
    case "failed":
      return "❌";
    case "canceled":
      return "⏹️";
    case "queued":
      return "⏳";
    default:
      return "📄";
  }
}

/**
 * Command Palette Component
 *
 * A modal command interface for quick access to all app features.
 * Uses cmdk for fuzzy search and keyboard navigation.
 */
export function CommandPalette({
  isOpen,
  onClose,
  jobs,
  onNavigate,
  onSubmitForm,
  onCancelJob,
  activeJobId,
  isMac = false,
  presets = [],
  onSelectPreset,
  onRestartTour,
}: CommandPaletteProps) {
  const [search, setSearch] = useState("");

  // Reset search when opening
  useEffect(() => {
    if (isOpen) {
      setSearch("");
    }
  }, [isOpen]);

  // Handle keyboard shortcuts within palette
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      // Close on Escape
      if (e.key === "Escape") {
        e.preventDefault();
        onClose();
        return;
      }
    },
    [onClose],
  );

  // Build command list
  const commands = useMemo((): CommandItem[] => {
    const list: CommandItem[] = [
      // Actions
      {
        id: "submit-scrape",
        label: "Submit Scrape Job",
        shortcut: isMac ? "⌘Enter" : "Ctrl+Enter",
        icon: "🔍",
        group: "actions",
        onSelect: () => {
          onSubmitForm("scrape");
          onClose();
        },
      },
      {
        id: "submit-crawl",
        label: "Submit Crawl Job",
        icon: "🕷️",
        group: "actions",
        onSelect: () => {
          onSubmitForm("crawl");
          onClose();
        },
      },
      {
        id: "submit-research",
        label: "Submit Research Job",
        icon: "📚",
        group: "actions",
        onSelect: () => {
          onSubmitForm("research");
          onClose();
        },
      },
      {
        id: "cancel-active",
        label: activeJobId ? "Cancel Active Job" : "Cancel Active Job (none)",
        icon: "⏹️",
        group: "actions",
        disabled: !activeJobId,
        onSelect: () => {
          if (activeJobId) {
            onCancelJob(activeJobId);
            onClose();
          }
        },
      },
      {
        id: "restart-tour",
        label: "Restart Onboarding Tour",
        icon: "🎯",
        group: "actions",
        onSelect: () => {
          onRestartTour?.();
          onClose();
        },
      },

      // Navigation
      {
        id: "nav-jobs",
        label: "Go to Jobs",
        shortcut: "G J",
        icon: "📋",
        group: "navigation",
        onSelect: () => {
          onNavigate("jobs");
          onClose();
        },
      },
      {
        id: "nav-results",
        label: "Go to Results",
        shortcut: "G R",
        icon: "📊",
        group: "navigation",
        onSelect: () => {
          onNavigate("results");
          onClose();
        },
      },
      {
        id: "nav-forms",
        label: "Go to Forms",
        shortcut: "G F",
        icon: "📝",
        group: "navigation",
        onSelect: () => {
          onNavigate("forms");
          onClose();
        },
      },
    ];

    // Add recent jobs (last 10)
    const recentJobs = jobs.slice(0, 10);
    for (const job of recentJobs) {
      list.push({
        id: `job-${job.id}`,
        label: `${getStatusIcon(job.status)} ${job.kind}: ${job.id.slice(0, 8)}`,
        icon: job.id === activeJobId ? "▶️" : undefined,
        group: "jobs",
        onSelect: () => {
          onNavigate("jobs");
          onClose();
        },
      });
    }

    // Add presets (built-in first, then custom)
    if (presets.length > 0 && onSelectPreset) {
      const sortedPresets = [...presets].sort((a, b) => {
        // Built-in first
        if (a.isBuiltIn && !b.isBuiltIn) return -1;
        if (!a.isBuiltIn && b.isBuiltIn) return 1;
        // Then alphabetically
        return a.name.localeCompare(b.name);
      });

      for (const preset of sortedPresets.slice(0, 20)) {
        list.push({
          id: `preset-${preset.id}`,
          label: `${preset.name} (${preset.jobType})`,
          icon: preset.icon,
          group: "presets",
          onSelect: () => {
            onSelectPreset(preset);
            onClose();
          },
        });
      }
    }

    return list;
  }, [
    jobs,
    presets,
    activeJobId,
    isMac,
    onNavigate,
    onSubmitForm,
    onCancelJob,
    onSelectPreset,
    onRestartTour,
    onClose,
  ]);

  if (!isOpen) return null;

  return (
    <div
      className="command-palette-overlay"
      onClick={(e) => {
        if (e.target === e.currentTarget) {
          onClose();
        }
      }}
      onKeyDown={handleKeyDown}
      role="dialog"
      aria-modal="true"
      aria-label="Command palette"
    >
      <Command
        className="command-palette"
        label="Command palette"
        value={search}
        onValueChange={setSearch}
        loop
      >
        <div className="command-palette-header">
          <Command.Input
            className="command-palette-input"
            placeholder="Type a command or search..."
            aria-label="Search commands"
          />
        </div>

        <Command.List className="command-palette-list">
          <Command.Empty className="command-palette-empty">
            No commands found.
          </Command.Empty>

          {/* Actions Group */}
          <Command.Group
            heading={
              <span className="command-palette-group-header">Actions</span>
            }
            className="command-palette-group"
          >
            {commands
              .filter((c) => c.group === "actions")
              .map((command) => (
                <Command.Item
                  key={command.id}
                  value={command.label}
                  disabled={command.disabled}
                  onSelect={command.onSelect}
                  className="command-palette-item"
                >
                  <span className="command-palette-item-icon">
                    {command.icon}
                  </span>
                  <span className="command-palette-item-label">
                    {command.label}
                  </span>
                  {command.shortcut && (
                    <kbd className="command-palette-shortcut">
                      {command.shortcut}
                    </kbd>
                  )}
                </Command.Item>
              ))}
          </Command.Group>

          {/* Navigation Group */}
          <Command.Group
            heading={
              <span className="command-palette-group-header">Navigation</span>
            }
            className="command-palette-group"
          >
            {commands
              .filter((c) => c.group === "navigation")
              .map((command) => (
                <Command.Item
                  key={command.id}
                  value={command.label}
                  disabled={command.disabled}
                  onSelect={command.onSelect}
                  className="command-palette-item"
                >
                  <span className="command-palette-item-icon">
                    {command.icon}
                  </span>
                  <span className="command-palette-item-label">
                    {command.label}
                  </span>
                  {command.shortcut && (
                    <kbd className="command-palette-shortcut">
                      {command.shortcut}
                    </kbd>
                  )}
                </Command.Item>
              ))}
          </Command.Group>

          {/* Presets Group */}
          {presets.length > 0 && (
            <Command.Group
              heading={
                <span className="command-palette-group-header">Presets</span>
              }
              className="command-palette-group"
            >
              {commands
                .filter((c) => c.group === "presets")
                .map((command) => (
                  <Command.Item
                    key={command.id}
                    value={command.label}
                    disabled={command.disabled}
                    onSelect={command.onSelect}
                    className="command-palette-item"
                  >
                    <span className="command-palette-item-icon">
                      {command.icon}
                    </span>
                    <span className="command-palette-item-label">
                      {command.label}
                    </span>
                  </Command.Item>
                ))}
            </Command.Group>
          )}

          {/* Jobs Group */}
          {jobs.length > 0 && (
            <Command.Group
              heading={
                <span className="command-palette-group-header">
                  Recent Jobs
                </span>
              }
              className="command-palette-group"
            >
              {commands
                .filter((c) => c.group === "jobs")
                .map((command) => (
                  <Command.Item
                    key={command.id}
                    value={command.label}
                    disabled={command.disabled}
                    onSelect={command.onSelect}
                    className="command-palette-item"
                  >
                    <span className="command-palette-item-label">
                      {command.label}
                    </span>
                  </Command.Item>
                ))}
            </Command.Group>
          )}
        </Command.List>

        <div className="command-palette-footer">
          <span className="command-palette-hint">
            <kbd>↑↓</kbd> to navigate
          </span>
          <span className="command-palette-hint">
            <kbd>↵</kbd> to select
          </span>
          <span className="command-palette-hint">
            <kbd>Esc</kbd> to close
          </span>
        </div>
      </Command>
    </div>
  );
}
