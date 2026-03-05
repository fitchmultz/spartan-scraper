# Workshop Outline (45–60 minutes)

## Audience

Engineers evaluating production-grade repo hardening and local-first CI discipline.

## Agenda

1. **Context (5 min)**
   - Project architecture and primary user journeys.

2. **CI and gate design (10 min)**
   - PR-required vs nightly/manual split.
   - Deterministic installs and bounded test concurrency.

3. **Security hardening walkthrough (10 min)**
   - Public audit scanner behavior.
   - WebSocket origin protection and localhost threat model.

4. **Hands-on lab (15–20 min)**
   - Run `make ci-pr`.
   - Run the WebSocket origin check.
   - Verify docs-driven onboarding.

5. **Debrief (10 min)**
   - Review remaining risks and roadmap.

## Success criteria

- Participants can run PR-equivalent gates locally.
- Participants can reproduce key security behavior checks.
- Participants can explain CI profile boundaries and trade-offs.

## Common failure modes

- Dirty git tree before `make ci-pr`
- Local env differences causing unexpected behavior
- Treating `ci-slow` variability as PR-gate regressions
