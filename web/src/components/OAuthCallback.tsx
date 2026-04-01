/**
 * Purpose: Render the o auth callback UI surface for the web operator experience.
 * Responsibilities: Define the component, its local view helpers, and the presentation logic owned by this file.
 * Scope: File-local UI behavior only; routing, persistence, and broader feature orchestration stay outside this file.
 * Usage: Import from the surrounding feature or route components that own this surface.
 * Invariants/Assumptions: Props and callbacks come from the surrounding feature contracts and should remain the single source of truth.
 */

import { useCallback, useEffect, useState } from "react";

interface OAuthCallbackProps {
  onSuccess?: () => void;
  onError?: (error: string) => void;
  redirectPath?: string;
}

export function OAuthCallback({
  onSuccess,
  onError,
  redirectPath = "/",
}: OAuthCallbackProps) {
  const [status, setStatus] = useState<
    "processing" | "success" | "error" | "popup"
  >("processing");
  const [error, setError] = useState<string>("");

  const exchangeOAuthCode = useCallback(async (code: string, state: string) => {
    const response = await fetch(
      `/v1/auth/oauth/callback?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`,
    );

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(
        errorData.error || `Token exchange failed: ${response.status}`,
      );
    }

    return response.json();
  }, []);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const code = params.get("code");
    const state = params.get("state");
    const errorParam = params.get("error");
    const errorDescription = params.get("error_description");

    // Check if this is a popup window
    const isPopup = window.opener !== null && window.opener !== window;

    if (errorParam) {
      const errorMsg = errorDescription || `OAuth error: ${errorParam}`;
      setStatus("error");
      setError(errorMsg);

      if (isPopup && window.opener) {
        window.opener.postMessage(
          { type: "OAUTH_ERROR", error: errorMsg, state },
          "*",
        );
        setTimeout(() => window.close(), 3000);
      } else if (onError) {
        onError(errorMsg);
      }
      return;
    }

    if (!code || !state) {
      const errorMsg = "Missing authorization code or state parameter";
      setStatus("error");
      setError(errorMsg);

      if (isPopup && window.opener) {
        window.opener.postMessage(
          { type: "OAUTH_ERROR", error: errorMsg, state },
          "*",
        );
        setTimeout(() => window.close(), 3000);
      } else if (onError) {
        onError(errorMsg);
      }
      return;
    }

    // Exchange code for token via API
    exchangeOAuthCode(code, state)
      .then(() => {
        setStatus("success");

        if (isPopup && window.opener) {
          window.opener.postMessage({ type: "OAUTH_SUCCESS", state }, "*");
          setTimeout(() => window.close(), 2000);
        } else {
          // Redirect mode
          if (onSuccess) {
            onSuccess();
          } else {
            window.location.href = redirectPath;
          }
        }
      })
      .catch((err) => {
        const errorMsg =
          err instanceof Error ? err.message : "Token exchange failed";
        setStatus("error");
        setError(errorMsg);

        if (isPopup && window.opener) {
          window.opener.postMessage(
            { type: "OAUTH_ERROR", error: errorMsg, state },
            "*",
          );
          setTimeout(() => window.close(), 3000);
        } else if (onError) {
          onError(errorMsg);
        }
      });
  }, [onSuccess, onError, redirectPath, exchangeOAuthCode]);

  const renderContent = () => {
    switch (status) {
      case "processing":
        return (
          <div style={{ textAlign: "center", padding: "40px" }}>
            <div
              style={{
                width: "40px",
                height: "40px",
                border: "3px solid rgba(255, 255, 255, 0.1)",
                borderTop: "3px solid var(--accent)",
                borderRadius: "50%",
                animation: "spin 1s linear infinite",
                margin: "0 auto 20px",
              }}
            />
            <style>{`
              @keyframes spin {
                0% { transform: rotate(0deg); }
                100% { transform: rotate(360deg); }
              }
            `}</style>
            <p>Completing authentication...</p>
          </div>
        );

      case "success":
        return (
          <div style={{ textAlign: "center", padding: "40px" }}>
            <div
              style={{
                width: "50px",
                height: "50px",
                borderRadius: "50%",
                background: "#4CAF50",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                margin: "0 auto 20px",
                fontSize: "24px",
              }}
            >
              ✓
            </div>
            <h3 style={{ marginBottom: "10px" }}>Authentication Successful</h3>
            <p style={{ color: "var(--text-secondary)" }}>
              {window.opener
                ? "You can close this window and return to the application."
                : "Redirecting you back to the application..."}
            </p>
          </div>
        );

      case "error":
        return (
          <div style={{ textAlign: "center", padding: "40px" }}>
            <div
              style={{
                width: "50px",
                height: "50px",
                borderRadius: "50%",
                background: "#f44336",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                margin: "0 auto 20px",
                fontSize: "24px",
              }}
            >
              ✕
            </div>
            <h3 style={{ marginBottom: "10px" }}>Authentication Failed</h3>
            <p style={{ color: "var(--text-secondary)", marginBottom: "20px" }}>
              {error}
            </p>
            {!window.opener && (
              <button
                type="button"
                onClick={() => {
                  window.location.href = redirectPath;
                }}
                style={{
                  padding: "10px 20px",
                  background: "var(--accent)",
                  border: "none",
                  borderRadius: "6px",
                  color: "white",
                  cursor: "pointer",
                }}
              >
                Return to Application
              </button>
            )}
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <div
      style={{
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        background: "var(--bg-primary)",
        color: "var(--text-primary)",
      }}
    >
      <div
        style={{
          background: "var(--bg-secondary)",
          borderRadius: "12px",
          padding: "20px",
          maxWidth: "400px",
          width: "90%",
          boxShadow: "0 4px 20px rgba(0, 0, 0, 0.3)",
        }}
      >
        {renderContent()}
      </div>
    </div>
  );
}

export default OAuthCallback;
