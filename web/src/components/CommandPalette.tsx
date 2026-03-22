/**
 * Purpose: Render the global fuzzy-searchable command palette for route jumps and high-frequency actions.
 * Responsibilities: Surface top-level navigation, automation-section navigation, preset selection, recent-job drill-down, and fast job actions.
 * Scope: Command palette presentation and command composition only.
 * Usage: Mount once from `App.tsx` and supply current jobs plus navigation/action callbacks.
 * Invariants/Assumptions: Commands must reach every major operator surface, recent-job entries drill into the actual job detail route, and closing the palette must not drop the selected action.
 */

import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import { Command } from "cmdk";
import { getJobStatusIcon } from "../lib/job-status";
import type { JobEntry } from "../types";
import type { JobPreset } from "../types/presets";

type CommandGroup =
  | "actions"
  | "navigation"
  | "automation"
  | "jobs"
  | "presets";

export type CommandPaletteProps = {
  isOpen: boolean;
  onClose: () => void;
  jobs: JobEntry[];
  onNavigateToPath: (path: string) => void;
  onSubmitForm: (formType: "scrape" | "crawl" | "research") => void;
  onCancelJob: (jobId: string) => void;
  activeJobId?: string;
  isMac?: boolean;
  presets?: JobPreset[];
  onSelectPreset?: (preset: JobPreset) => void;
  onRestartTour?: () => void;
};

type CommandItem = {
  id: string;
  label: string;
  shortcut?: string;
  icon?: string;
  group: CommandGroup;
  disabled?: boolean;
  onSelect: () => void;
};

const GROUP_ORDER: Array<{ key: CommandGroup; heading: string }> = [
  { key: "actions", heading: "Actions" },
  { key: "navigation", heading: "Navigation" },
  { key: "automation", heading: "Automation" },
  { key: "presets", heading: "Presets" },
  { key: "jobs", heading: "Recent Jobs" },
];

const ROUTE_COMMANDS = [
  {
    id: "nav-jobs",
    label: "Open Jobs",
    path: "/jobs",
    shortcut: "G J",
    icon: "📋",
    group: "navigation" as const,
  },
  {
    id: "nav-new-job",
    label: "Create Job",
    path: "/jobs/new",
    shortcut: "G F",
    icon: "📝",
    group: "navigation" as const,
  },
  {
    id: "nav-templates",
    label: "Open Templates",
    path: "/templates",
    icon: "🧩",
    group: "navigation" as const,
  },
  {
    id: "nav-settings",
    label: "Open Settings",
    path: "/settings",
    icon: "⚙️",
    group: "navigation" as const,
  },
  {
    id: "nav-automation-batches",
    label: "Open Automation / Batches",
    path: "/automation/batches",
    icon: "📦",
    group: "automation" as const,
  },
  {
    id: "nav-automation-chains",
    label: "Open Automation / Chains",
    path: "/automation/chains",
    icon: "🔗",
    group: "automation" as const,
  },
  {
    id: "nav-automation-watches",
    label: "Open Automation / Watches",
    path: "/automation/watches",
    icon: "👀",
    group: "automation" as const,
  },
  {
    id: "nav-automation-exports",
    label: "Open Automation / Exports",
    path: "/automation/exports",
    icon: "📤",
    group: "automation" as const,
  },
  {
    id: "nav-automation-webhooks",
    label: "Open Automation / Webhooks",
    path: "/automation/webhooks",
    icon: "🪝",
    group: "automation" as const,
  },
] as const;

export function CommandPalette({
  isOpen,
  onClose,
  jobs,
  onNavigateToPath,
  onSubmitForm,
  onCancelJob,
  activeJobId,
  isMac = false,
  presets = [],
  onSelectPreset,
  onRestartTour,
}: CommandPaletteProps) {
  const [search, setSearch] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (isOpen) {
      setSearch("");
    }
  }, [isOpen]);

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const raf = window.requestAnimationFrame(() => {
      inputRef.current?.focus({ preventScroll: true });
      inputRef.current?.select();
    });

    return () => window.cancelAnimationFrame(raf);
  }, [isOpen]);

  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        onClose();
      }
    },
    [onClose],
  );

  const latestResultJob =
    jobs.find((job) => job.status === "succeeded") ?? jobs[0] ?? null;

  const commands = useMemo((): CommandItem[] => {
    const list: CommandItem[] = [
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
          if (!activeJobId) {
            return;
          }
          onCancelJob(activeJobId);
          onClose();
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
      {
        id: "nav-latest-results",
        label: latestResultJob
          ? `Open Latest Results (${latestResultJob.id.slice(0, 8)})`
          : "Open Latest Results",
        icon: "📊",
        group: "navigation",
        disabled: !latestResultJob,
        onSelect: () => {
          if (!latestResultJob) {
            return;
          }
          onNavigateToPath(`/jobs/${latestResultJob.id}`);
          onClose();
        },
      },
    ];

    for (const route of ROUTE_COMMANDS) {
      list.push({
        id: route.id,
        label: route.label,
        shortcut: "shortcut" in route ? route.shortcut : undefined,
        icon: route.icon,
        group: route.group,
        onSelect: () => {
          onNavigateToPath(route.path);
          onClose();
        },
      });
    }

    for (const preset of [...presets]
      .sort((a, b) =>
        a.isBuiltIn === b.isBuiltIn
          ? a.name.localeCompare(b.name)
          : a.isBuiltIn
            ? -1
            : 1,
      )
      .slice(0, 20)) {
      list.push({
        id: `preset-${preset.id}`,
        label: `${preset.name} (${preset.jobType})`,
        icon: preset.icon,
        group: "presets",
        onSelect: () => {
          onSelectPreset?.(preset);
          onClose();
        },
      });
    }

    for (const job of jobs.slice(0, 10)) {
      list.push({
        id: `job-${job.id}`,
        label: `${getJobStatusIcon(job.status)} ${job.kind}: ${job.id.slice(0, 8)}`,
        icon: job.id === activeJobId ? "▶️" : undefined,
        group: "jobs",
        onSelect: () => {
          onNavigateToPath(`/jobs/${job.id}`);
          onClose();
        },
      });
    }

    return list;
  }, [
    activeJobId,
    isMac,
    jobs,
    latestResultJob,
    onCancelJob,
    onClose,
    onNavigateToPath,
    onRestartTour,
    onSelectPreset,
    onSubmitForm,
    presets,
  ]);

  if (!isOpen) {
    return null;
  }

  return (
    <div
      className="command-palette-overlay"
      onClick={(event) => {
        if (event.target === event.currentTarget) {
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
            ref={inputRef}
            className="command-palette-input"
            placeholder="Type a command or search..."
            aria-label="Search commands"
            autoFocus
          />
        </div>

        <Command.List className="command-palette-list">
          <Command.Empty className="command-palette-empty">
            No commands found.
          </Command.Empty>

          {GROUP_ORDER.map(({ key, heading }) => {
            const groupCommands = commands.filter(
              (command) => command.group === key,
            );
            if (groupCommands.length === 0) {
              return null;
            }

            return (
              <Command.Group
                key={key}
                heading={
                  <span className="command-palette-group-header">
                    {heading}
                  </span>
                }
                className="command-palette-group"
              >
                {groupCommands.map((command) => (
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
                    {command.shortcut ? (
                      <kbd className="command-palette-shortcut">
                        {command.shortcut}
                      </kbd>
                    ) : null}
                  </Command.Item>
                ))}
              </Command.Group>
            );
          })}
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
