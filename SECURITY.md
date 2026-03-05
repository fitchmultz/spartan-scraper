# Security Policy

## Reporting Vulnerabilities

**Do NOT open public issues for security vulnerabilities.**

If you discover a security vulnerability, report it privately through one of these channels:

1. Preferred: Open a private GitHub security advisory draft in this repository.
2. Email: [security@spartan-scraper.dev](mailto:security@spartan-scraper.dev)

Please include as much detail as possible:

- A description of the vulnerability
- Steps to reproduce the issue
- Impact of the vulnerability
- Affected versions
- Any potential mitigations you have identified

We will acknowledge your report within 48 hours (if reasonable) and work with you to understand and address the issue.

## Supported Versions

Currently, Spartan Scraper is in pre-1.0 development. Only the main branch receives security fixes.

As the project matures and versioned releases are established, this policy will be updated to reflect supported version ranges.

## Response Process

Our security response process follows these steps:

1. **Acknowledgment**: We will acknowledge receipt of your vulnerability report within 48 hours (if reasonable).

2. **Assessment**: We will investigate and validate the vulnerability, typically within 7 days.

3. **Fix Development**: We will develop and test a fix. The timeline depends on severity:
   - Critical: Aim for 7-14 days
   - High: Aim for 14-30 days
   - Medium: Aim for 30-60 days
   - Low: Scheduled in regular release cycle

4. **Coordinated Disclosure**: We will work with you on a coordinated disclosure timeline. Fixes are released before public disclosure.

5. **Credit**: With your permission, we will credit you in the release notes and security advisory.

6. **Public Announcement**: After the fix is deployed, we will publish a security advisory with details about the vulnerability and the fix.

## What We Consider a Vulnerability

We consider the following as security vulnerabilities that should be reported privately:

- Remote code execution
- Authentication or authorization bypass
- Data exposure (including secrets, auth vault, or sensitive user data)
- Denial of service vulnerabilities
- Cross-site scripting (XSS)
- Cross-site request forgery (CSRF)
- SQL injection or similar injection attacks
- Information disclosure of sensitive data
- Privilege escalation
- Any issue that compromises the security or privacy of users

## Non-Vulnerability Issues

The following are NOT considered security vulnerabilities for this project:

- **Robots.txt compliance**: Robots.txt handling is ignored by default and can be enabled explicitly via `--respect-robots` or `RESPECT_ROBOTS_TXT=true`. Default behavior is a product policy choice, not a vulnerability.
- **Target website rate limiting**: Rate limiting to target websites is configurable via `.env`. Improper configuration is a deployment concern, not a vulnerability.
- **Feature requests or non-security bugs**: These should be reported through normal channels (see CONTRIBUTING.md for bug reporting guidelines).

## Project Status

Spartan Scraper is a volunteer-run, pre-1.0 project. We do not provide:

- SLA guarantees for response or fix timelines
- Monetary bounties for vulnerability reports
- 24/7 security support

However, we are committed to addressing security issues responsibly and maintaining the trust of our users.

## Security Best Practices

Users of Spartan Scraper should follow these security best practices:

- Keep the software updated to the latest version
- Use environment variables (`.env`) for sensitive configuration
- Do not commit secrets or API keys to the repository
- Review and test any custom extraction templates or pipeline JavaScript
- Be cautious when scraping sites that require authentication
- Use appropriate rate limiting to avoid overloading target servers
- Review auth vault contents in `.data/auth_vault.json` regularly

## Additional Information

For general questions or non-security issues, please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on reporting bugs and getting help.

---

Thank you for helping keep Spartan Scraper secure!
