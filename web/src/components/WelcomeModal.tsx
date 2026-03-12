/**
 * Welcome Modal Component
 *
 * First-time user welcome modal with value proposition and
 * options to start the tour, try a demo, or skip onboarding.
 *
 * @module components/WelcomeModal
 */

import { useCallback, useEffect, useState } from "react";

export interface WelcomeModalProps {
  /** Whether the modal is open */
  isOpen: boolean;
  /** Callback when user wants to start the tour */
  onStartTour: () => void;
  /** Callback when user wants to try the demo */
  onTryDemo?: () => void;
  /** Callback when user skips onboarding */
  onSkip: () => void;
}

/**
 * Welcome Modal Component
 *
 * Displays a friendly welcome message for first-time users with
 * clear options to start the guided tour, try a demo, or skip.
 *
 * @example
 * ```tsx
 * <WelcomeModal
 *   isOpen={shouldShowWelcome}
 *   onStartTour={startOnboarding}
 *   onTryDemo={runDemoJob}
 *   onSkip={skipOnboarding}
 * />
 * ```
 */
export function WelcomeModal({
  isOpen,
  onStartTour,
  onTryDemo,
  onSkip,
}: WelcomeModalProps) {
  const [isAnimating, setIsAnimating] = useState(false);

  // Handle escape key
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onSkip();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onSkip]);

  // Animation on mount
  useEffect(() => {
    if (isOpen) {
      const timer = setTimeout(() => setIsAnimating(true), 10);
      return () => clearTimeout(timer);
    }
    setIsAnimating(false);
  }, [isOpen]);

  const handleStartTour = useCallback(() => {
    setIsAnimating(false);
    onStartTour();
  }, [onStartTour]);

  const handleTryDemo = useCallback(() => {
    setIsAnimating(false);
    onTryDemo?.();
  }, [onTryDemo]);

  const handleSkip = useCallback(() => {
    setIsAnimating(false);
    onSkip();
  }, [onSkip]);

  if (!isOpen) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="welcome-title"
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0, 0, 0, 0.8)",
        backdropFilter: "blur(8px)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 1000,
        padding: "20px",
        animation: "welcome-fade-in 0.3s ease",
      }}
      onClick={(e) => {
        if (e.target === e.currentTarget) {
          handleSkip();
        }
      }}
      onKeyDown={(e) => {
        if (e.key === "Escape") {
          handleSkip();
        }
      }}
    >
      <div
        style={{
          background: "var(--panel, #1a1a24)",
          border: "1px solid var(--stroke, rgba(255,255,255,0.1))",
          borderRadius: 24,
          padding: "40px",
          maxWidth: 560,
          width: "100%",
          boxShadow: "0 24px 64px var(--shadow, rgba(0,0,0,0.5))",
          transform: isAnimating
            ? "translateY(0) scale(1)"
            : "translateY(20px) scale(0.95)",
          opacity: isAnimating ? 1 : 0,
          transition: "transform 0.3s ease, opacity 0.3s ease",
        }}
      >
        {/* Header with gradient accent */}
        <div
          style={{
            background:
              "linear-gradient(135deg, var(--accent, #ffb700), var(--accent-strong, #ff9500))",
            margin: "-40px -40px 24px",
            padding: "32px 40px",
            borderRadius: "24px 24px 0 0",
            position: "relative",
            overflow: "hidden",
          }}
        >
          {/* Decorative circles */}
          <div
            style={{
              position: "absolute",
              top: -20,
              right: -20,
              width: 100,
              height: 100,
              borderRadius: "50%",
              background: "rgba(255,255,255,0.1)",
            }}
          />
          <div
            style={{
              position: "absolute",
              bottom: -30,
              left: 40,
              width: 60,
              height: 60,
              borderRadius: "50%",
              background: "rgba(255,255,255,0.08)",
            }}
          />

          <h1
            id="welcome-title"
            style={{
              margin: 0,
              fontSize: "1.75rem",
              fontWeight: 700,
              color: "#1a1200",
              position: "relative",
            }}
          >
            Welcome to Spartan Scraper
          </h1>
          <p
            style={{
              margin: "8px 0 0",
              fontSize: "1rem",
              color: "rgba(26, 18, 0, 0.8)",
              position: "relative",
            }}
          >
            Your web scraping and research companion
          </p>
        </div>

        {/* Value Propositions */}
        <div style={{ marginBottom: 32 }}>
          <ValueProp
            icon="🔍"
            title="Scrape, Crawl & Research"
            description="Extract data from single pages, crawl entire sites, or conduct multi-source research with AI-powered analysis."
          />
          <ValueProp
            icon="⚡"
            title="Built-in Templates & Presets"
            description="Get started quickly with templates for common patterns: blogs, SPAs, e-commerce, and more."
          />
          <ValueProp
            icon="📦"
            title="Flexible Export Options"
            description="Export to Markdown, CSV, or JSON with customizable pipelines and transformations."
          />
        </div>

        {/* Action Buttons */}
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            gap: 12,
          }}
        >
          <button
            type="button"
            onClick={handleStartTour}
            style={{
              padding: "14px 24px",
              fontSize: "1rem",
              fontWeight: 600,
              background:
                "linear-gradient(135deg, var(--accent, #ffb700), var(--accent-strong, #ff9500))",
              color: "#1a1200",
              border: "none",
              borderRadius: 12,
              cursor: "pointer",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              gap: 8,
              transition: "transform 0.15s ease, box-shadow 0.15s ease",
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.transform = "translateY(-2px)";
              e.currentTarget.style.boxShadow =
                "0 8px 24px rgba(255, 183, 0, 0.3)";
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.transform = "translateY(0)";
              e.currentTarget.style.boxShadow = "none";
            }}
          >
            <span>🚀</span>
            Take a Quick Tour
          </button>

          {onTryDemo && (
            <button
              type="button"
              onClick={handleTryDemo}
              style={{
                padding: "14px 24px",
                fontSize: "1rem",
                fontWeight: 500,
                background: "transparent",
                color: "var(--text, #fff)",
                border: "1px solid var(--stroke, rgba(255,255,255,0.1))",
                borderRadius: 12,
                cursor: "pointer",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                gap: 8,
                transition: "background 0.15s ease",
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.background = "rgba(255,255,255,0.05)";
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.background = "transparent";
              }}
            >
              <span>🎮</span>
              Try Demo First
            </button>
          )}

          <button
            type="button"
            onClick={handleSkip}
            style={{
              padding: "12px 24px",
              fontSize: "0.85rem",
              background: "transparent",
              color: "var(--muted, #a0a0a8)",
              border: "none",
              cursor: "pointer",
              transition: "color 0.15s ease",
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.color = "var(--text, #fff)";
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.color = "var(--muted, #a0a0a8)";
            }}
          >
            Skip for now — you can always restart the tour from the Command
            Palette
          </button>
        </div>

        {/* Keyboard hint */}
        <div
          style={{
            marginTop: 24,
            paddingTop: 24,
            borderTop: "1px solid var(--stroke, rgba(255,255,255,0.1))",
            textAlign: "center",
            fontSize: "0.75rem",
            color: "var(--muted, #a0a0a8)",
          }}
        >
          Press{" "}
          <kbd
            style={{
              background: "rgba(255,255,255,0.1)",
              padding: "2px 8px",
              borderRadius: 4,
              fontFamily: "inherit",
              fontSize: "0.7rem",
            }}
          >
            Esc
          </kbd>{" "}
          to close,{" "}
          <kbd
            style={{
              background: "rgba(255,255,255,0.1)",
              padding: "2px 8px",
              borderRadius: 4,
              fontFamily: "inherit",
              fontSize: "0.7rem",
            }}
          >
            ?
          </kbd>{" "}
          for keyboard shortcuts (when focus is outside text inputs)
        </div>
      </div>
    </div>
  );
}

/**
 * Individual value proposition item.
 */
function ValueProp({
  icon,
  title,
  description,
}: {
  icon: string;
  title: string;
  description: string;
}) {
  return (
    <div
      style={{
        display: "flex",
        gap: 16,
        alignItems: "flex-start",
        marginBottom: 16,
      }}
    >
      <span
        style={{
          fontSize: "1.5rem",
          flexShrink: 0,
          width: 40,
          height: 40,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          background: "rgba(255,255,255,0.05)",
          borderRadius: 10,
        }}
      >
        {icon}
      </span>
      <div>
        <h3
          style={{
            margin: "0 0 4px",
            fontSize: "0.95rem",
            fontWeight: 600,
            color: "var(--text, #fff)",
          }}
        >
          {title}
        </h3>
        <p
          style={{
            margin: 0,
            fontSize: "0.85rem",
            color: "var(--muted, #a0a0a8)",
            lineHeight: 1.5,
          }}
        >
          {description}
        </p>
      </div>
    </div>
  );
}
