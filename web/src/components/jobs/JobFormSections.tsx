import type { ReactNode } from "react";

interface JobFormIntroProps {
  title: string;
  description: string;
  children: ReactNode;
  actions: ReactNode;
}

interface JobFormAdvancedSectionProps {
  title: string;
  description: string;
  children: ReactNode;
  defaultOpen?: boolean;
}

export function JobFormIntro({
  title,
  description,
  children,
  actions,
}: JobFormIntroProps) {
  return (
    <section className="panel job-workflow-form__primary">
      <div className="job-workflow-form__intro">
        <div className="job-workflow-form__eyebrow">Primary Workflow</div>
        <h2>{title}</h2>
        <p>{description}</p>
      </div>
      <div className="job-workflow-form__body">{children}</div>
      <div className="job-workflow-form__actions">{actions}</div>
    </section>
  );
}

export function JobFormAdvancedSection({
  title,
  description,
  children,
  defaultOpen = false,
}: JobFormAdvancedSectionProps) {
  return (
    <details className="job-workflow-form__advanced" open={defaultOpen}>
      <summary>
        <div>
          <span>{title}</span>
          <small>{description}</small>
        </div>
      </summary>
      <div className="job-workflow-form__advanced-body">{children}</div>
    </details>
  );
}
