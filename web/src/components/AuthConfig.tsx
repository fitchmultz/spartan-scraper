/**
 * Auth Config Component
 *
 * Reusable authentication configuration UI shared across all job forms.
 * Handles auth profile selection, basic auth, custom headers, cookies, query params,
 * and headless login flow configuration (URL, selectors, credentials).
 *
 * @module AuthConfig
 */

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
}

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
}: AuthConfigProps) {
  return (
    <>
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
                value={loginUserSelector}
                onChange={(event) => setLoginUserSelector(event.target.value)}
                placeholder="#email"
              />
            </label>
            <label>
              Pass Selector
              <input
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
                value={loginUser}
                onChange={(event) => setLoginUser(event.target.value)}
                placeholder="you@example.com"
              />
            </label>
            <label>
              Password
              <input
                type="password"
                value={loginPass}
                onChange={(event) => setLoginPass(event.target.value)}
                placeholder="•••••••"
              />
            </label>
          </div>
        </div>
      </details>
    </>
  );
}
