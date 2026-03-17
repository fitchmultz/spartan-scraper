/**
 * Purpose: Render the guided wizard runtime step using the existing runtime/auth/capture field groups.
 * Responsibilities: Group execution, browser/capture, and authentication settings into labeled subpanels and show blocking validation summaries when present.
 * Scope: Guided job wizard runtime step only.
 * Usage: Render from `JobSubmissionContainer` while guided mode is on the `runtime` step.
 * Invariants/Assumptions: Runtime controls continue to flow through the shared `FormController`, profile options are already loaded, and disabled browser-only controls remain visible for context.
 */

import { AuthConfig } from "../../AuthConfig";
import { BrowserExecutionControls } from "../../BrowserExecutionControls";
import { DeviceSelector } from "../../DeviceSelector";
import { NetworkInterceptConfig } from "../../NetworkInterceptConfig";
import { ScreenshotConfig } from "../../ScreenshotConfig";
import type { DeviceEmulation } from "../../../api";
import type {
  FormController,
  ProfileOption,
} from "../../../hooks/useFormState";

interface RuntimeStepProps {
  form: FormController;
  profiles: ProfileOption[];
  device: DeviceEmulation | null;
  setDevice: (value: DeviceEmulation | null) => void;
  errors: string[];
  inputPrefix: "scrape" | "crawl" | "research";
}

export function RuntimeStep({
  form,
  profiles,
  device,
  setDevice,
  errors,
  inputPrefix,
}: RuntimeStepProps) {
  return (
    <section className="panel job-wizard__panel">
      <div className="job-wizard__panel-header">
        <div className="job-workflow__eyebrow">Runtime</div>
        <h2>Choose how the job should execute</h2>
        <p>
          Browser mode, timeout, authentication, request shaping, device
          emulation, and capture settings live together here.
        </p>
      </div>

      {errors.length > 0 ? (
        <div className="job-wizard__error-summary" role="alert">
          <strong>Fix these before continuing:</strong>
          <ul>
            {errors.map((error) => (
              <li key={error}>{error}</li>
            ))}
          </ul>
        </div>
      ) : null}

      <div className="job-wizard__subpanel-grid">
        <section className="job-wizard__subpanel">
          <h3>Execution</h3>
          <BrowserExecutionControls
            headless={form.headless}
            setHeadless={form.setHeadless}
            usePlaywright={form.usePlaywright}
            setUsePlaywright={form.setUsePlaywright}
            timeoutSeconds={form.timeoutSeconds}
            setTimeoutSeconds={form.setTimeoutSeconds}
          />
        </section>

        <section className="job-wizard__subpanel">
          <h3>Browser and capture</h3>
          <ScreenshotConfig
            enabled={form.screenshotEnabled}
            setEnabled={form.setScreenshotEnabled}
            fullPage={form.screenshotFullPage}
            setFullPage={form.setScreenshotFullPage}
            format={form.screenshotFormat}
            setFormat={form.setScreenshotFormat}
            quality={form.screenshotQuality}
            setQuality={form.setScreenshotQuality}
            width={form.screenshotWidth}
            setWidth={form.setScreenshotWidth}
            height={form.screenshotHeight}
            setHeight={form.setScreenshotHeight}
            disabled={!form.headless}
            inputPrefix={inputPrefix}
          />
          <DeviceSelector
            device={device}
            onChange={setDevice}
            disabled={!form.headless}
          />
          <NetworkInterceptConfig
            enabled={form.interceptEnabled}
            setEnabled={form.setInterceptEnabled}
            urlPatterns={form.interceptURLPatterns}
            setURLPatterns={form.setInterceptURLPatterns}
            resourceTypes={form.interceptResourceTypes}
            setResourceTypes={form.setInterceptResourceTypes}
            captureRequestBody={form.interceptCaptureRequestBody}
            setCaptureRequestBody={form.setInterceptCaptureRequestBody}
            captureResponseBody={form.interceptCaptureResponseBody}
            setCaptureResponseBody={form.setInterceptCaptureResponseBody}
            maxBodySize={form.interceptMaxBodySize}
            setMaxBodySize={form.setInterceptMaxBodySize}
            maxEntries={form.interceptMaxEntries}
            setMaxEntries={form.setInterceptMaxEntries}
            disabled={!form.headless}
            inputPrefix={inputPrefix}
          />
        </section>

        <section className="job-wizard__subpanel">
          <h3>Authentication and request shaping</h3>
          <AuthConfig
            authProfile={form.authProfile}
            setAuthProfile={form.setAuthProfile}
            authBasic={form.authBasic}
            setAuthBasic={form.setAuthBasic}
            headersRaw={form.headersRaw}
            setHeadersRaw={form.setHeadersRaw}
            cookiesRaw={form.cookiesRaw}
            setCookiesRaw={form.setCookiesRaw}
            queryRaw={form.queryRaw}
            setQueryRaw={form.setQueryRaw}
            proxyUrl={form.proxyUrl}
            setProxyUrl={form.setProxyUrl}
            proxyUsername={form.proxyUsername}
            setProxyUsername={form.setProxyUsername}
            proxyPassword={form.proxyPassword}
            setProxyPassword={form.setProxyPassword}
            proxyRegion={form.proxyRegion}
            setProxyRegion={form.setProxyRegion}
            proxyRequiredTags={form.proxyRequiredTags}
            setProxyRequiredTags={form.setProxyRequiredTags}
            proxyExcludeProxyIds={form.proxyExcludeProxyIds}
            setProxyExcludeProxyIds={form.setProxyExcludeProxyIds}
            loginUrl={form.loginUrl}
            setLoginUrl={form.setLoginUrl}
            loginUserSelector={form.loginUserSelector}
            setLoginUserSelector={form.setLoginUserSelector}
            loginPassSelector={form.loginPassSelector}
            setLoginPassSelector={form.setLoginPassSelector}
            loginSubmitSelector={form.loginSubmitSelector}
            setLoginSubmitSelector={form.setLoginSubmitSelector}
            loginUser={form.loginUser}
            setLoginUser={form.setLoginUser}
            loginPass={form.loginPass}
            setLoginPass={form.setLoginPass}
            profiles={profiles}
          />
        </section>
      </div>
    </section>
  );
}
