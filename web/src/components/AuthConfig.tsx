/**
 * Purpose: Render the shared authentication and session-override controls used across operator job forms.
 * Responsibilities: Edit auth profile selection, direct credentials, header/cookie/query overrides, proxy settings, and OAuth discovery inputs in one reusable surface.
 * Scope: Web-form configuration UI only; submitting jobs and persisting auth state remain parent responsibilities.
 * Usage: Mount inside scrape, crawl, research, and batch forms with controlled state setters from `useFormState`.
 * Invariants/Assumptions: Parent components own all field state, OAuth discovery remains optional, and discovery failures should surface as operator-facing toast feedback instead of blocking the rest of the form.
 */

import type { CSSProperties } from "react";
import { useToast } from "./toast";

interface OAuth2Config {
  flowType: "authorization_code" | "client_credentials" | "device_code";
  clientId: string;
  clientSecret?: string;
  discoveryUrl?: string;
  authorizeUrl?: string;
  tokenUrl: string;
  scopes: string[];
  usePkce: boolean;
  redirectUri?: string;
}

interface AuthConfigProps {
  authProfile: string;
  setAuthProfile: (value: string) => void;
  authBasic: string;
  setAuthBasic: (value: string) => void;
  headersRaw: string;
  setHeadersRaw: (value: string) => void;
  cookiesRaw: string;
  setCookiesRaw: (value: string) => void;
  queryRaw: string;
  setQueryRaw: (value: string) => void;
  proxyUrl: string;
  setProxyUrl: (value: string) => void;
  proxyUsername: string;
  setProxyUsername: (value: string) => void;
  proxyPassword: string;
  setProxyPassword: (value: string) => void;
  proxyRegion: string;
  setProxyRegion: (value: string) => void;
  proxyRequiredTags: string;
  setProxyRequiredTags: (value: string) => void;
  proxyExcludeProxyIds: string;
  setProxyExcludeProxyIds: (value: string) => void;
  loginUrl: string;
  setLoginUrl: (value: string) => void;
  loginUserSelector: string;
  setLoginUserSelector: (value: string) => void;
  loginPassSelector: string;
  setLoginPassSelector: (value: string) => void;
  loginSubmitSelector: string;
  setLoginSubmitSelector: (value: string) => void;
  loginUser: string;
  setLoginUser: (value: string) => void;
  loginPass: string;
  setLoginPass: (value: string) => void;
  profiles: Array<{ name: string; parents: string[] }>;
  // OAuth props
  oauthConfig?: OAuth2Config;
  setOauthConfig?: (config: OAuth2Config | undefined) => void;
  onOAuthInitiate?: () => void;
}

// OAuth provider presets
const OAUTH_PROVIDER_PRESETS: Record<
  string,
  {
    name: string;
    discoveryUrl?: string;
    authorizeUrl?: string;
    tokenUrl?: string;
    defaultScopes: string[];
    requiresPkce: boolean;
  }
> = {
  google: {
    name: "Google",
    discoveryUrl:
      "https://accounts.google.com/.well-known/openid-configuration",
    defaultScopes: ["openid", "email", "profile"],
    requiresPkce: true,
  },
  github: {
    name: "GitHub",
    authorizeUrl: "https://github.com/login/oauth/authorize",
    tokenUrl: "https://github.com/login/oauth/access_token",
    defaultScopes: ["read:user", "repo"],
    requiresPkce: false,
  },
  microsoft: {
    name: "Microsoft",
    discoveryUrl:
      "https://login.microsoftonline.com/common/v2.0/.well-known/openid-configuration",
    defaultScopes: ["openid", "email", "profile"],
    requiresPkce: true,
  },
  auth0: {
    name: "Auth0",
    discoveryUrl: "", // User needs to provide their tenant URL
    defaultScopes: ["openid", "profile", "email"],
    requiresPkce: true,
  },
  okta: {
    name: "Okta",
    discoveryUrl: "", // User needs to provide their domain
    defaultScopes: ["openid", "profile", "email"],
    requiresPkce: true,
  },
  custom: {
    name: "Custom",
    defaultScopes: [],
    requiresPkce: true,
  },
};

const maskedSecretStyle = {
  WebkitTextSecurity: "disc",
} as CSSProperties;

export function AuthConfig({
  authProfile,
  setAuthProfile,
  authBasic,
  setAuthBasic,
  headersRaw,
  setHeadersRaw,
  cookiesRaw,
  setCookiesRaw,
  queryRaw,
  setQueryRaw,
  proxyUrl,
  setProxyUrl,
  proxyUsername,
  setProxyUsername,
  proxyPassword,
  setProxyPassword,
  proxyRegion,
  setProxyRegion,
  proxyRequiredTags,
  setProxyRequiredTags,
  proxyExcludeProxyIds,
  setProxyExcludeProxyIds,
  loginUrl,
  setLoginUrl,
  loginUserSelector,
  setLoginUserSelector,
  loginPassSelector,
  setLoginPassSelector,
  loginSubmitSelector,
  setLoginSubmitSelector,
  loginUser,
  setLoginUser,
  loginPass,
  setLoginPass,
  profiles,
  oauthConfig,
  setOauthConfig,
  onOAuthInitiate,
}: AuthConfigProps) {
  const toast = useToast();
  const hasOAuth = oauthConfig !== undefined && setOauthConfig !== undefined;

  const handleProviderChange = (provider: string) => {
    if (!setOauthConfig) return;

    const preset = OAUTH_PROVIDER_PRESETS[provider];
    if (!preset) {
      setOauthConfig(undefined);
      return;
    }

    setOauthConfig({
      flowType: "authorization_code",
      clientId: "",
      clientSecret: preset.requiresPkce ? undefined : "",
      discoveryUrl: preset.discoveryUrl,
      authorizeUrl: preset.authorizeUrl,
      tokenUrl: preset.tokenUrl || "",
      scopes: [...preset.defaultScopes],
      usePkce: preset.requiresPkce,
      redirectUri: "http://localhost:8741/oauth/callback",
    });
  };

  const handleDiscoverOIDC = async () => {
    if (!oauthConfig?.discoveryUrl) return;

    try {
      const response = await fetch("/v1/auth/oauth/discover", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ discovery_url: oauthConfig.discoveryUrl }),
      });

      if (!response.ok) {
        throw new Error("OIDC discovery failed");
      }

      const metadata = await response.json();

      if (setOauthConfig) {
        setOauthConfig({
          ...oauthConfig,
          authorizeUrl: metadata.authorization_endpoint,
          tokenUrl: metadata.token_endpoint,
        });
      }
    } catch (error) {
      toast.show({
        tone: "error",
        title: "OIDC discovery failed",
        description:
          error instanceof Error ? error.message : "Unknown discovery error.",
      });
    }
  };

  const selectedProvider = oauthConfig
    ? Object.entries(OAUTH_PROVIDER_PRESETS).find(
        ([, preset]) =>
          preset.discoveryUrl === oauthConfig.discoveryUrl ||
          preset.authorizeUrl === oauthConfig.authorizeUrl,
      )?.[0] || "custom"
    : "";

  return (
    <div data-tour="auth-profiles">
      <label htmlFor="auth-profile">Auth Profile</label>
      <select
        id="auth-profile"
        value={authProfile}
        onChange={(event) => setAuthProfile(event.target.value)}
      >
        <option value="">None</option>
        {profiles.map((p) => (
          <option key={p.name} value={p.name}>
            {p.name}{" "}
            {p.parents.length > 0 ? `(extends: ${p.parents.join(", ")})` : ""}
          </option>
        ))}
      </select>
      <label htmlFor="auth-basic" style={{ marginTop: 12 }}>
        Basic auth (user:pass)
      </label>
      <input
        id="auth-basic"
        autoComplete="off"
        value={authBasic}
        onChange={(event) => setAuthBasic(event.target.value)}
      />
      <label htmlFor="headers-raw" style={{ marginTop: 12 }}>
        Extra headers (one per line: Key: Value)
      </label>
      <textarea
        id="headers-raw"
        rows={3}
        value={headersRaw}
        onChange={(event) => setHeadersRaw(event.target.value)}
      />
      <label htmlFor="cookies-raw" style={{ marginTop: 12 }}>
        Cookies (one per line: name=value)
      </label>
      <textarea
        id="cookies-raw"
        rows={2}
        value={cookiesRaw}
        onChange={(event) => setCookiesRaw(event.target.value)}
        placeholder="session_id=abc123&#10;auth_token=xyz789"
      />
      <label htmlFor="query-raw" style={{ marginTop: 12 }}>
        Query params (one per line: key=value)
      </label>
      <textarea
        id="query-raw"
        rows={2}
        value={queryRaw}
        onChange={(event) => setQueryRaw(event.target.value)}
        placeholder="api_key=your_key&#10;version=v1"
      />
      <label htmlFor="proxy-url" style={{ marginTop: 12 }}>
        Direct proxy URL
      </label>
      <input
        id="proxy-url"
        autoComplete="off"
        value={proxyUrl}
        onChange={(event) => setProxyUrl(event.target.value)}
        placeholder="http://proxy.example:8080"
      />
      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12 }}>
        <div>
          <label htmlFor="proxy-username" style={{ marginTop: 12 }}>
            Proxy username
          </label>
          <input
            id="proxy-username"
            autoComplete="off"
            value={proxyUsername}
            onChange={(event) => setProxyUsername(event.target.value)}
          />
        </div>
        <div>
          <label htmlFor="proxy-password" style={{ marginTop: 12 }}>
            Proxy password
          </label>
          <input
            id="proxy-password"
            type="password"
            autoComplete="off"
            value={proxyPassword}
            onChange={(event) => setProxyPassword(event.target.value)}
          />
        </div>
      </div>
      <p style={{ margin: "8px 0 0", opacity: 0.8, fontSize: 12 }}>
        Direct proxy settings bypass the loaded proxy pool for this request.
      </p>
      <label htmlFor="proxy-region" style={{ marginTop: 12 }}>
        Preferred proxy region
      </label>
      <input
        id="proxy-region"
        autoComplete="off"
        value={proxyRegion}
        onChange={(event) => setProxyRegion(event.target.value)}
        placeholder="us-east"
      />
      <label htmlFor="proxy-required-tags" style={{ marginTop: 12 }}>
        Required proxy tags (comma-separated)
      </label>
      <input
        id="proxy-required-tags"
        autoComplete="off"
        value={proxyRequiredTags}
        onChange={(event) => setProxyRequiredTags(event.target.value)}
        placeholder="residential, sticky"
      />
      <label htmlFor="proxy-exclude-proxy-ids" style={{ marginTop: 12 }}>
        Excluded proxy IDs (comma-separated)
      </label>
      <input
        id="proxy-exclude-proxy-ids"
        autoComplete="off"
        value={proxyExcludeProxyIds}
        onChange={(event) => setProxyExcludeProxyIds(event.target.value)}
        placeholder="proxy-east, proxy-west"
      />
      <p style={{ margin: "8px 0 0", opacity: 0.8, fontSize: 12 }}>
        Proxy-pool selection hints require the loaded global pool and cannot be
        combined with a direct proxy override.
      </p>

      {hasOAuth && (
        <details style={{ marginTop: 12 }}>
          <summary
            style={{
              cursor: "pointer",
              marginBottom: "8px",
              color: "var(--accent)",
            }}
          >
            OAuth 2.0 Configuration
          </summary>
          <div
            style={{
              marginTop: "12px",
              padding: "12px",
              borderRadius: "12px",
              background: "rgba(0, 0, 0, 0.25)",
            }}
          >
            <label htmlFor="oauth-provider">Provider</label>
            <select
              id="oauth-provider"
              value={selectedProvider}
              onChange={(event) => handleProviderChange(event.target.value)}
            >
              <option value="">None</option>
              {Object.entries(OAUTH_PROVIDER_PRESETS).map(([key, preset]) => (
                <option key={key} value={key}>
                  {preset.name}
                </option>
              ))}
            </select>

            {oauthConfig && (
              <>
                <label htmlFor="oauth-flow-type" style={{ marginTop: 8 }}>
                  Flow Type
                </label>
                <select
                  id="oauth-flow-type"
                  value={oauthConfig.flowType}
                  onChange={(event) =>
                    setOauthConfig?.({
                      ...oauthConfig,
                      flowType: event.target.value as OAuth2Config["flowType"],
                    })
                  }
                >
                  <option value="authorization_code">Authorization Code</option>
                  <option value="client_credentials">Client Credentials</option>
                </select>

                <label htmlFor="oauth-client-id" style={{ marginTop: 8 }}>
                  Client ID
                </label>
                <input
                  id="oauth-client-id"
                  type="text"
                  value={oauthConfig.clientId}
                  onChange={(event) =>
                    setOauthConfig?.({
                      ...oauthConfig,
                      clientId: event.target.value,
                    })
                  }
                />

                {(!oauthConfig.usePkce || oauthConfig.clientSecret) && (
                  <>
                    <label
                      htmlFor="oauth-client-secret"
                      style={{ marginTop: 8 }}
                    >
                      Client Secret
                    </label>
                    <input
                      id="oauth-client-secret"
                      type="text"
                      autoComplete="off"
                      value={oauthConfig.clientSecret || ""}
                      onChange={(event) =>
                        setOauthConfig?.({
                          ...oauthConfig,
                          clientSecret: event.target.value || undefined,
                        })
                      }
                      spellCheck={false}
                      style={maskedSecretStyle}
                    />
                  </>
                )}

                {oauthConfig.flowType === "authorization_code" && (
                  <>
                    <label
                      style={{
                        marginTop: 8,
                        display: "flex",
                        alignItems: "center",
                        gap: "8px",
                      }}
                    >
                      <input
                        type="checkbox"
                        checked={oauthConfig.usePkce}
                        onChange={(event) =>
                          setOauthConfig?.({
                            ...oauthConfig,
                            usePkce: event.target.checked,
                            clientSecret: event.target.checked
                              ? undefined
                              : oauthConfig.clientSecret,
                          })
                        }
                      />
                      Use PKCE (recommended for public clients)
                    </label>

                    <label
                      htmlFor="oauth-discovery-url"
                      style={{ marginTop: 8 }}
                    >
                      Discovery URL (OIDC)
                    </label>
                    <div style={{ display: "flex", gap: "8px" }}>
                      <input
                        id="oauth-discovery-url"
                        type="url"
                        value={oauthConfig.discoveryUrl || ""}
                        onChange={(event) =>
                          setOauthConfig?.({
                            ...oauthConfig,
                            discoveryUrl: event.target.value || undefined,
                          })
                        }
                        placeholder="https://example.com/.well-known/openid-configuration"
                        style={{ flex: 1 }}
                      />
                      <button
                        type="button"
                        onClick={handleDiscoverOIDC}
                        disabled={!oauthConfig.discoveryUrl}
                      >
                        Discover
                      </button>
                    </div>

                    <label
                      htmlFor="oauth-authorize-url"
                      style={{ marginTop: 8 }}
                    >
                      Authorization URL
                    </label>
                    <input
                      id="oauth-authorize-url"
                      type="url"
                      value={oauthConfig.authorizeUrl || ""}
                      onChange={(event) =>
                        setOauthConfig?.({
                          ...oauthConfig,
                          authorizeUrl: event.target.value || undefined,
                        })
                      }
                      placeholder="https://example.com/oauth/authorize"
                    />

                    <label
                      htmlFor="oauth-redirect-uri"
                      style={{ marginTop: 8 }}
                    >
                      Redirect URI
                    </label>
                    <input
                      id="oauth-redirect-uri"
                      type="url"
                      value={oauthConfig.redirectUri || ""}
                      onChange={(event) =>
                        setOauthConfig?.({
                          ...oauthConfig,
                          redirectUri: event.target.value || undefined,
                        })
                      }
                      placeholder="http://localhost:8741/oauth/callback"
                    />
                  </>
                )}

                <label htmlFor="oauth-token-url" style={{ marginTop: 8 }}>
                  Token URL
                </label>
                <input
                  id="oauth-token-url"
                  type="url"
                  value={oauthConfig.tokenUrl}
                  onChange={(event) =>
                    setOauthConfig?.({
                      ...oauthConfig,
                      tokenUrl: event.target.value,
                    })
                  }
                  placeholder="https://example.com/oauth/token"
                />

                <label htmlFor="oauth-scopes" style={{ marginTop: 8 }}>
                  Scopes (space-separated)
                </label>
                <input
                  id="oauth-scopes"
                  type="text"
                  value={oauthConfig.scopes.join(" ")}
                  onChange={(event) =>
                    setOauthConfig?.({
                      ...oauthConfig,
                      scopes: event.target.value
                        .split(" ")
                        .filter((s) => s.trim()),
                    })
                  }
                  placeholder="openid email profile"
                />

                {onOAuthInitiate &&
                  oauthConfig.flowType === "authorization_code" && (
                    <button
                      type="button"
                      onClick={onOAuthInitiate}
                      disabled={!oauthConfig.clientId || !oauthConfig.tokenUrl}
                      style={{ marginTop: 12 }}
                    >
                      Authenticate with OAuth
                    </button>
                  )}
              </>
            )}
          </div>
        </details>
      )}

      <details>
        <summary
          style={{
            cursor: "pointer",
            marginBottom: "8px",
            color: "var(--accent)",
          }}
        >
          Login Flow Configuration (Headless Auth)
        </summary>
        <div
          style={{
            marginTop: "12px",
            padding: "12px",
            borderRadius: "12px",
            background: "rgba(0, 0, 0, 0.25)",
          }}
        >
          <label htmlFor="login-url">Login URL</label>
          <input
            id="login-url"
            value={loginUrl}
            onChange={(event) => setLoginUrl(event.target.value)}
            placeholder="https://example.com/login"
          />
          <div className="row" style={{ marginTop: "12px" }}>
            <label>
              User Selector
              <input
                autoComplete="off"
                name="login-user"
                value={loginUserSelector}
                onChange={(event) => setLoginUserSelector(event.target.value)}
                placeholder="#email"
              />
            </label>
            <label>
              Pass Selector
              <input
                autoComplete="off"
                name="login-pass-selector"
                value={loginPassSelector}
                onChange={(event) => setLoginPassSelector(event.target.value)}
                placeholder="#password"
              />
            </label>
          </div>
          <div className="row" style={{ marginTop: "12px" }}>
            <label>
              Submit Selector
              <input
                value={loginSubmitSelector}
                onChange={(event) => setLoginSubmitSelector(event.target.value)}
                placeholder="button[type=submit]"
              />
            </label>
          </div>
          <div className="row" style={{ marginTop: "12px" }}>
            <label>
              Username
              <input
                type="text"
                autoComplete="username"
                name="login-username"
                value={loginUser}
                onChange={(event) => setLoginUser(event.target.value)}
                placeholder="you@example.com"
              />
            </label>
            <label>
              Password
              <input
                type="password"
                autoComplete="current-password"
                name="login-password"
                value={loginPass}
                onChange={(event) => setLoginPass(event.target.value)}
                placeholder="•••••••"
              />
            </label>
          </div>
        </div>
      </details>
    </div>
  );
}
