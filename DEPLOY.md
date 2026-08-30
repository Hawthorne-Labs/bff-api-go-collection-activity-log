# Deploy / runtime

**Production ECS `bff-activity-log` runs this Go BFF** (cutover 2026-08-30).

- Do **not** deploy `mvp/bff-api-python-collection-activity-log` for normal releases.
- Catalog: `jenkins-ci/services.json` → `bff-activity-log` (`type=bff-go`, `blockedVariants: ["python"]`).
- Agent/operator registry: `docs/ai/bff-runtime-cutover.md`.

```bash
cd jenkins-ci/deploy-console
./deploy-console build-deploy -env prod -service bff-activity-log -branch feat/r1-opt -release <slug>
```
