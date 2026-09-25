"""The six-stage triage pipeline (architecture section 3).

A user's description of symptoms moves through every stage below before any
text reaches them. The pipeline exists because passing a medical question
straight to a language model is not safe: each stage narrows what the model
is allowed to be wrong about.

======  ==========================  ====  ==================================
Stage   Module                      Week  What it does
======  ==========================  ====  ==================================
1       ``extraction``              10    Free text (or a whisper.cpp
                                          transcript) to structured symptoms
2       ``classification``          10    Structured symptoms to Green /
                                          Yellow / Red emergency level
3       ``retrieval``               11    Pull matching first-aid articles
                                          from the knowledge base
4       ``prompt``                  11    Assemble model input: symptoms,
                                          retrieved articles, and the
                                          patient's own triage history
5       ``inference``               9     Call MedGemma via ``app.ai.llm``
6       ``safety``                  12    Validate before anything is shown
======  ==========================  ====  ==================================

Two properties hold across the whole pipeline:

**Conservative failure.** If any stage cannot complete confidently, the
result is a conservative pre-approved response plus a doctor
recommendation -- never a guess. A wrong "you are fine" is far more
dangerous here than an unnecessary escalation.

**Per-patient memory.** Stage 4 reads that patient's recent ``triage_logs``
and ``symptom_reports``, so "fever yesterday, breathing difficulty today"
is evaluated as one story. This context is scoped to a single patient and
never shared across accounts.
"""
