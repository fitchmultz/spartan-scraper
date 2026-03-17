/**
 * Purpose: Render the guided wizard extraction step by reusing the existing extraction, AI, webhook, and research-agentic field groups.
 * Responsibilities: Group template/processor settings, AI extraction controls, research-agentic controls, and webhook delivery options into a guided layout.
 * Scope: Guided job wizard extraction step only.
 * Usage: Render from `JobSubmissionContainer` while guided mode is on the `extraction` step.
 * Invariants/Assumptions: Extraction settings continue to flow through the shared `FormController`, research-only agentic settings stay hidden for non-research jobs, and validation errors are summarized at the top of the step.
 */

import { AIExtractSection } from "../../AIExtractSection";
import { PipelineOptions } from "../../PipelineOptions";
import { ResearchAgenticSection } from "../../ResearchAgenticSection";
import { WebhookConfig } from "../../WebhookConfig";
import type { FormController } from "../../../hooks/useFormState";
import type { JobType } from "../../../types/presets";

interface ExtractionStepProps {
  activeTab: JobType;
  form: FormController;
  errors: string[];
}

export function ExtractionStep({
  activeTab,
  form,
  errors,
}: ExtractionStepProps) {
  return (
    <section className="panel job-wizard__panel">
      <div className="job-wizard__panel-header">
        <div className="job-workflow__eyebrow">Extraction</div>
        <h2>Define what the job should produce</h2>
        <p>
          Templates, validators, processors, AI extraction, research-specific
          agentic settings, and delivery hooks belong in this stage.
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
          <h3>Template and processing</h3>
          <PipelineOptions
            extractTemplate={form.extractTemplate}
            setExtractTemplate={form.setExtractTemplate}
            extractValidate={form.extractValidate}
            setExtractValidate={form.setExtractValidate}
            preProcessors={form.preProcessors}
            setPreProcessors={form.setPreProcessors}
            postProcessors={form.postProcessors}
            setPostProcessors={form.setPostProcessors}
            transformers={form.transformers}
            setTransformers={form.setTransformers}
            incremental={
              activeTab === "research" ? undefined : form.incremental
            }
            setIncremental={
              activeTab === "research" ? undefined : form.setIncremental
            }
            inputPrefix={activeTab}
          />
        </section>

        <section className="job-wizard__subpanel">
          <h3>AI extraction</h3>
          <AIExtractSection
            enabled={form.aiExtractEnabled}
            setEnabled={form.setAIExtractEnabled}
            mode={form.aiExtractMode}
            setMode={form.setAIExtractMode}
            prompt={form.aiExtractPrompt}
            setPrompt={form.setAIExtractPrompt}
            schemaText={form.aiExtractSchema}
            setSchemaText={form.setAIExtractSchema}
            fields={form.aiExtractFields}
            setFields={form.setAIExtractFields}
          />
        </section>

        {activeTab === "research" ? (
          <section className="job-wizard__subpanel">
            <h3>Agentic research</h3>
            <ResearchAgenticSection
              enabled={form.agenticResearchEnabled}
              setEnabled={form.setAgenticResearchEnabled}
              instructions={form.agenticResearchInstructions}
              setInstructions={form.setAgenticResearchInstructions}
              maxRounds={form.agenticResearchMaxRounds}
              setMaxRounds={form.setAgenticResearchMaxRounds}
              maxFollowUpUrls={form.agenticResearchMaxFollowUpUrls}
              setMaxFollowUpUrls={form.setAgenticResearchMaxFollowUpUrls}
            />
          </section>
        ) : null}

        <section className="job-wizard__subpanel">
          <h3>Webhook delivery</h3>
          <WebhookConfig
            webhookUrl={form.webhookUrl}
            setWebhookUrl={form.setWebhookUrl}
            webhookEvents={form.webhookEvents}
            setWebhookEvents={form.setWebhookEvents}
            webhookSecret={form.webhookSecret}
            setWebhookSecret={form.setWebhookSecret}
            inputPrefix={activeTab}
          />
        </section>
      </div>
    </section>
  );
}
