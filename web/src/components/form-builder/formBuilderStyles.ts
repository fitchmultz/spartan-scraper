/**
 * Purpose: Hold the inline style sheet string used by the form builder UI surface.
 * Responsibilities: Export the scoped CSS text that keeps the extracted form builder sections styled without moving to a broader stylesheet in this batch.
 * Scope: Form builder styles only; other component styles stay in their existing files.
 * Usage: Render inside `<style>{FORM_BUILDER_STYLES}</style>` from `FormBuilder.tsx`.
 * Invariants/Assumptions: Class names stay aligned with the current form builder markup and remain intentionally local to this surface.
 */

export const FORM_BUILDER_STYLES = `
  .form-builder {
    background: var(--bg-primary, #1a1a2e);
    border-radius: 12px;
    padding: 24px;
    max-width: 800px;
    margin: 0 auto;
  }

  .form-builder-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
    padding-bottom: 16px;
    border-bottom: 1px solid var(--border-color, #2d2d44);
  }

  .form-builder-header h2 {
    margin: 0;
    color: var(--text-primary, #eaeaea);
  }

  .close-button {
    background: none;
    border: none;
    font-size: 24px;
    color: var(--text-secondary, #a0a0a0);
    cursor: pointer;
    padding: 0;
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
  }

  .close-button:hover {
    background: var(--bg-hover, #2d2d44);
    color: var(--text-primary, #eaeaea);
  }

  .section {
    margin-bottom: 24px;
  }

  .section h3 {
    margin: 0 0 12px 0;
    color: var(--text-primary, #eaeaea);
    font-size: 16px;
  }

  .url-input-group {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .url-input {
    flex: 1;
    min-width: 250px;
    padding: 10px 14px;
    border: 1px solid var(--border-color, #2d2d44);
    border-radius: 6px;
    background: var(--bg-secondary, #16213e);
    color: var(--text-primary, #eaeaea);
    font-size: 14px;
  }

  .form-type-select {
    padding: 10px 14px;
    border: 1px solid var(--border-color, #2d2d44);
    border-radius: 6px;
    background: var(--bg-secondary, #16213e);
    color: var(--text-primary, #eaeaea);
    font-size: 14px;
    min-width: 120px;
  }

  .detect-button,
  .execute-button {
    padding: 10px 20px;
    border: none;
    border-radius: 6px;
    background: var(--accent-primary, #0f3460);
    color: white;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.2s;
  }

  .detect-button:hover:not(:disabled),
  .execute-button:hover:not(:disabled) {
    background: var(--accent-hover, #1a4a7a);
  }

  .detect-button:disabled,
  .execute-button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .error-message {
    padding: 12px 16px;
    background: rgba(231, 76, 60, 0.1);
    border: 1px solid rgba(231, 76, 60, 0.3);
    border-radius: 6px;
    color: #e74c3c;
    margin-bottom: 16px;
  }

  .forms-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .form-card {
    padding: 12px 16px;
    background: var(--bg-secondary, #16213e);
    border: 2px solid transparent;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .form-card:hover {
    border-color: var(--accent-primary, #0f3460);
  }

  .form-card.selected {
    border-color: var(--accent-primary, #0f3460);
    background: rgba(15, 52, 96, 0.2);
  }

  .form-card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 4px;
  }

  .form-type {
    font-weight: 600;
    color: var(--accent-primary, #4a9eff);
    text-transform: capitalize;
  }

  .form-score {
    font-size: 12px;
    color: var(--text-secondary, #a0a0a0);
  }

  .form-selector {
    font-family: monospace;
    font-size: 12px;
    color: var(--text-secondary, #a0a0a0);
    margin-bottom: 4px;
  }

  .form-fields-count {
    font-size: 12px;
    color: var(--text-muted, #666);
  }

  .fields-table {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .field-row {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 12px;
    background: var(--bg-secondary, #16213e);
    border-radius: 6px;
  }

  .field-info {
    flex: 0 0 200px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .field-name {
    font-weight: 500;
    color: var(--text-primary, #eaeaea);
  }

  .required {
    color: #e74c3c;
    margin-left: 4px;
  }

  .field-type {
    font-size: 12px;
    color: var(--accent-primary, #4a9eff);
    text-transform: capitalize;
  }

  .field-placeholder {
    font-size: 11px;
    color: var(--text-muted, #666);
  }

  .field-input {
    flex: 1;
    padding: 8px 12px;
    border: 1px solid var(--border-color, #2d2d44);
    border-radius: 4px;
    background: var(--bg-primary, #1a1a2e);
    color: var(--text-primary, #eaeaea);
    font-size: 14px;
  }

  .submit-options {
    margin-top: 16px;
    padding: 16px;
    background: var(--bg-secondary, #16213e);
    border-radius: 6px;
  }

  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--text-primary, #eaeaea);
    cursor: pointer;
  }

  .wait-for-input {
    margin-top: 12px;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .wait-for-input label {
    color: var(--text-secondary, #a0a0a0);
    font-size: 14px;
  }

  .wait-for-input input {
    flex: 1;
    padding: 8px 12px;
    border: 1px solid var(--border-color, #2d2d44);
    border-radius: 4px;
    background: var(--bg-primary, #1a1a2e);
    color: var(--text-primary, #eaeaea);
    font-size: 14px;
  }

  .execute-button {
    margin-top: 16px;
    width: 100%;
    padding: 12px;
  }

  .result-section {
    padding: 16px;
    border-radius: 6px;
  }

  .result-section.success {
    background: rgba(46, 204, 113, 0.1);
    border: 1px solid rgba(46, 204, 113, 0.3);
  }

  .result-section.error {
    background: rgba(231, 76, 60, 0.1);
    border: 1px solid rgba(231, 76, 60, 0.3);
  }

  .result-content {
    color: var(--text-primary, #eaeaea);
  }

  .result-status {
    font-weight: 600;
    margin-bottom: 12px;
  }

  .result-section.success .result-status {
    color: #2ecc71;
  }

  .result-section.error .result-status {
    color: #e74c3c;
  }

  .filled-fields,
  .errors,
  .page-url {
    margin-top: 8px;
  }

  .filled-fields ul,
  .errors ul {
    margin: 4px 0;
    padding-left: 20px;
  }

  .filled-fields li,
  .errors li {
    margin: 2px 0;
  }

  .errors li {
    color: #e74c3c;
  }
`;
