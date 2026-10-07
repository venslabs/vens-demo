# Frozen runs

GitHub Actions artifacts expire after 30 days, and re-running the workflow does not
reproduce them — `demo.yml` pins the mutable tag `python:3.12-slim`, which moves, and
nothing pins the Trivy database. So the runs worth keeping are copied here, one dated
folder per run, append-only.

Each folder carries a `MANIFEST.json` with the run id, the resolved image digest, the
vens version and model that produced it, the counts, and a `sha256` per file.

| Run | Date | Image | vens | Model | Note |
|---|---|---|---|---|---|
| [2026-09-23](2026-09-23/) | 2026-09-23 | `python:3.12-slim` @ `2f17fc04` | v0.5.0 | `openai/gpt-5.4-mini`, batch 10 | the `before` for venslabs/vens#322 |
