"""Text-to-speech -- localized Nepali narration of first-aid guidance.

Audio output is an accessibility feature, not a nicety. Someone performing
first aid has both hands occupied and cannot read a screen, and a
meaningful share of the users this platform targets are more comfortable
listening than reading.

Planned surface:

``synthesize(text, *, language) -> bytes``
    Renders validated guidance to audio. Runs in the Celery
    ``ai.synthesize_speech`` task and caches by content hash in Redis --
    the same handful of first-aid instructions are narrated repeatedly, so
    synthesizing each one once is worth the cache.

Only safety-validated text is ever synthesized: audio is harder for a user
to skim critically than text, so unvalidated model output must never reach
this module.

Configuration: ``tts_engine``, ``tts_model_path``, ``tts_output_format``.
"""
