/**
 * Purpose: Provide route-local containers for the automation, settings, and setup-recovery workflows.
 * Responsibilities: Render sectioned automation and settings workspaces, keep route-local derived state close to each surface, and preserve route help plus inventory/status summaries.
 * Scope: Automation/settings/setup route presentation only; route parsing and global shell orchestration stay in `App.tsx`.
 * Usage: Re-export through `AppRoutes.tsx` and render from the application shell after route selection.
 * Invariants/Assumptions: Automation and settings deep-link shapes remain stable, route-local counters stay derived from existing feature surfaces, and setup-required mode remains a read-only recovery surface.
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import { shouldShowSettingsOverviewPanel } from "../../lib/settings-overview";
import type {
  ExportSchedulePromotionSeed,
  WatchPromotionSeed,
} from "../../types/promotion";
import { InfoSections } from "../InfoSections";
import { AutomationLayout } from "../automation/AutomationLayout";
import {
  getAutomationPath,
  type AutomationSection,
} from "../automation/automationSections";
import { AutomationSubnav } from "../automation/AutomationSubnav";
import { BatchContainer } from "../batches/BatchContainer";
import { ChainContainer } from "../chains/ChainContainer";
import { ActionEmptyState } from "../ActionEmptyState";
import { ExportScheduleContainer } from "../export-schedules/ExportScheduleContainer";
import { PipelineJSEditor } from "../pipeline-js/PipelineJSEditor";
import { ProxyPoolStatusPanel } from "../ProxyPoolStatusPanel";
import { RenderProfileEditor } from "../render-profiles";
import { RetentionStatusPanel } from "../RetentionStatusPanel";
import { RouteHelpPanel } from "../RouteHelpPanel";
import { SettingsOverviewPanel } from "../SettingsOverviewPanel";
import { RouteHeader } from "../shell/ShellPrimitives";
import {
  getSettingsPath,
  SETTINGS_SECTION_META,
  type SettingsSectionId,
} from "../settings/settingsSections";
import { SettingsSubnav } from "../settings/SettingsSubnav";
import { WatchContainer } from "../watches/WatchContainer";
import { WebhookDeliveryContainer } from "../webhooks/WebhookDeliveryContainer";
import {
  scrollWindowToTop,
  type AutomationRouteProps,
  type SettingsRouteProps,
} from "./routeTypes";

export function AutomationRoute({
  section,
  promotionSeed = null,
  formState,
  profiles,
  loading,
  aiStatus = null,
  routeHelp,
  onClearPromotionSeed,
  navigate,
  onRefreshJobs,
}: AutomationRouteProps) {
  const watchPromotionSeed =
    section === "watches" && promotionSeed?.kind === "watch"
      ? (promotionSeed as WatchPromotionSeed)
      : null;
  const exportPromotionSeed =
    section === "exports" && promotionSeed?.kind === "export-schedule"
      ? (promotionSeed as ExportSchedulePromotionSeed)
      : null;

  const renderSection = useCallback(
    (activeSection: AutomationSection): ReactNode => {
      switch (activeSection) {
        case "batches":
          return (
            <BatchContainer
              formState={formState}
              profiles={profiles}
              loading={loading}
            />
          );
        case "chains":
          return <ChainContainer onChainSubmit={onRefreshJobs} />;
        case "watches":
          return (
            <WatchContainer
              promotionSeed={watchPromotionSeed}
              onClearPromotionSeed={onClearPromotionSeed}
              onOpenSourceJob={(jobId) => navigate(`/jobs/${jobId}`)}
            />
          );
        case "exports":
          return (
            <ExportScheduleContainer
              aiStatus={aiStatus}
              promotionSeed={exportPromotionSeed}
              onClearPromotionSeed={onClearPromotionSeed}
              onOpenSourceJob={(jobId) => navigate(`/jobs/${jobId}`)}
            />
          );
        case "webhooks":
          return <WebhookDeliveryContainer />;
      }
    },
    [
      aiStatus,
      exportPromotionSeed,
      formState,
      loading,
      navigate,
      onClearPromotionSeed,
      onRefreshJobs,
      profiles,
      watchPromotionSeed,
    ],
  );

  return (
    <div className="route-stack">
      <RouteHeader
        title="Automation"
        subnav={
          <div data-tour="automation-subnav">
            <AutomationSubnav
              activeSection={section}
              onSectionChange={(nextSection) =>
                navigate(getAutomationPath(nextSection))
              }
            />
          </div>
        }
      />

      <section data-tour="automation-hub">
        <AutomationLayout
          activeSection={section}
          renderSection={renderSection}
        />
      </section>

      <RouteHelpPanel routeKey="automation" {...routeHelp} />
    </div>
  );
}

export function SettingsRoute({
  section,
  path,
  health,
  profiles,
  schedules,
  crawlStates,
  crawlStatesPage,
  crawlStatesTotal,
  jobsTotal,
  routeHelp,
  onNavigate,
  onRefreshHealth,
  onCrawlStatesPageChange,
}: SettingsRouteProps) {
  const [renderProfileCount, setRenderProfileCount] = useState<number | null>(
    null,
  );
  const [pipelineScriptCount, setPipelineScriptCount] = useState<number | null>(
    null,
  );

  const showSettingsOverview = useMemo(
    () =>
      shouldShowSettingsOverviewPanel({
        isSettingsRoute: true,
        setupRequired: false,
        jobsTotal,
        profilesCount: profiles.length,
        schedulesCount: schedules.length,
        crawlStatesTotal,
        renderProfileCount,
        pipelineScriptCount,
        proxyStatus: health?.components?.proxy_pool?.status,
        retentionStatus: health?.components?.retention?.status,
      }),
    [
      crawlStatesTotal,
      health?.components?.proxy_pool?.status,
      health?.components?.retention?.status,
      jobsTotal,
      pipelineScriptCount,
      profiles.length,
      renderProfileCount,
      schedules.length,
    ],
  );

  useEffect(() => {
    if (!path) {
      return;
    }

    scrollWindowToTop();
  }, [path]);

  const scrollToSettingsSection = useCallback(
    (nextSection: SettingsSectionId) => {
      if (nextSection === section) {
        scrollWindowToTop();
        return;
      }

      onNavigate(getSettingsPath(nextSection));
    },
    [onNavigate, section],
  );

  const renderSection = useCallback(
    (activeSection: SettingsSectionId): ReactNode => {
      switch (activeSection) {
        case "authoring":
          return (
            <div className="settings-route__section-stack">
              <section className="panel">
                <RenderProfileEditor
                  aiStatus={health?.components?.ai ?? null}
                  onInventoryChange={setRenderProfileCount}
                />
              </section>

              <section className="panel">
                <PipelineJSEditor
                  aiStatus={health?.components?.ai ?? null}
                  onInventoryChange={setPipelineScriptCount}
                />
              </section>
            </div>
          );
        case "inventory":
          return (
            <InfoSections
              profiles={profiles}
              schedules={schedules}
              crawlStates={crawlStates}
              crawlStatesPage={crawlStatesPage}
              crawlStatesTotal={crawlStatesTotal}
              crawlStatesPerPage={100}
              onCrawlStatesPageChange={onCrawlStatesPageChange}
              onCreateJob={() => onNavigate("/jobs/new")}
              onOpenAutomation={() => onNavigate("/automation/batches")}
              onOpenJobs={() => onNavigate("/jobs")}
            />
          );
        case "operations":
          return (
            <div className="settings-route__section-stack">
              <ProxyPoolStatusPanel
                health={health}
                onNavigate={onNavigate}
                onRefreshHealth={onRefreshHealth}
              />
              <RetentionStatusPanel
                health={health}
                onNavigate={onNavigate}
                onRefreshHealth={onRefreshHealth}
                onCreateJob={() => onNavigate("/jobs/new")}
                onOpenAutomation={() => onNavigate("/automation/batches")}
              />
            </div>
          );
      }
    },
    [
      crawlStates,
      crawlStatesPage,
      crawlStatesTotal,
      health,
      onCrawlStatesPageChange,
      onNavigate,
      onRefreshHealth,
      profiles,
      schedules,
    ],
  );

  return (
    <div className="route-stack">
      <RouteHeader
        title="Settings"
        subnav={
          <SettingsSubnav
            activeSection={section}
            onSectionChange={scrollToSettingsSection}
          />
        }
      />

      <div data-tour="settings-workspace" className="settings-route">
        <section
          id={SETTINGS_SECTION_META[section].elementId}
          className="settings-route__section"
          aria-labelledby={`settings-route-${section}-title`}
        >
          <div className="settings-route__section-header">
            <div className="settings-route__section-eyebrow">
              {SETTINGS_SECTION_META[section].label}
            </div>
            <h2 id={`settings-route-${section}-title`}>
              {SETTINGS_SECTION_META[section].title}
            </h2>
            <p>{SETTINGS_SECTION_META[section].description}</p>
          </div>

          {renderSection(section)}
        </section>

        {showSettingsOverview ? (
          <SettingsOverviewPanel
            onCreateJob={() => onNavigate("/jobs/new")}
            onOpenJobs={() => onNavigate("/jobs")}
          />
        ) : null}
      </div>

      <RouteHelpPanel routeKey="settings" {...routeHelp} />
    </div>
  );
}

export function SetupRequiredRoute({
  health,
}: {
  health: import("../../api").HealthResponse | null;
}) {
  return (
    <div className="route-stack">
      <RouteHeader
        title="Setup required"
        description="Spartan is running in guided recovery mode so the issue is visible in-product instead of only in terminal output."
      />

      <ActionEmptyState
        eyebrow="Guided recovery"
        title={health?.setup?.title ?? "Setup required"}
        description={
          health?.setup?.message ??
          "Resolve the setup issue, then restart the server."
        }
      />
    </div>
  );
}
