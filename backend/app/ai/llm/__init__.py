"""MedGemma client -- the clinical triage model (Week 9).

MedGemma is a medically-tuned open model, chosen over a general-purpose LLM
because triage answers must be grounded in clinical language rather than
plausible-sounding prose. It is reached over HTTP so the same code path
serves an Ollama container in development and a hosted inference endpoint in
production; only ``MEDGEMMA_BASE_URL`` changes.

Planned surface:

``generate(prompt, *, context) -> Completion``
    One inference call, returning the raw completion plus token usage and
    latency for the metrics dashboards (architecture section 12).

Configuration lives in ``Settings``: ``medgemma_model``,
``medgemma_base_url``, ``medgemma_api_key``, ``medgemma_timeout_seconds``,
``medgemma_max_tokens``, and ``medgemma_temperature`` -- which defaults to
0.0, because the same described symptoms must not yield different medical
guidance on two consecutive runs.

The model's output is never returned to a user directly. It always passes
through ``app.ai.triage.safety`` first.
"""
