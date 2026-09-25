---
title: SwasthyaSetu MedGemma
emoji: 🩺
colorFrom: green
colorTo: blue
sdk: gradio
sdk_version: 6.28.0
python_version: "3.12"
app_file: app.py
pinned: false
suggested_hardware: a10g-small
models:
  - google/medgemma-1.5-4b-it
---

# SwasthyaSetu · MedGemma Space

Serves `google/medgemma-1.5-4b-it` for the SwasthyaSetu demo. The backend calls
the `/generate` API; the page itself is a chat UI for testing the model by hand.

The front matter above is the Space configuration. Full setup steps are in the
main repository at `docs/deployment.md`.
