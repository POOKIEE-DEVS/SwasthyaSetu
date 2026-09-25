"""Curated medical content and the retrieval that grounds triage answers.

Guidance is retrieved from vetted sources, never generated freeform, so
every answer traces back to something a clinician approved:

- WHO first-aid and emergency-care guidelines
- Red Cross first-aid protocols
- Nepal Health Protocol references
- The project's own emergency handbook, maintained by its medical
  contributors

Content is stored in the ``articles`` table with ``source_name`` and
``source_url`` required, which is what makes a response auditable after the
fact.

Retrieval (Week 11) starts as Postgres full-text search over that table.
That is a deliberate first choice rather than a placeholder: it needs no
extra infrastructure, the corpus is small and hand-curated, and keyword
matching is debuggable in a way embedding similarity is not -- when a
clinician asks why an article surfaced, there is an answer. Embeddings
(``pgvector``) become worth their cost once the corpus is large enough that
vocabulary mismatch is the real failure mode.
"""
