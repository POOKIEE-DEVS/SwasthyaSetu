"""Speech-to-text -- whisper.cpp, optimized for on-device Nepali input.

whisper.cpp rather than a cloud transcription API for two reasons that both
matter to this product: it runs without connectivity, which is precisely
when a first-aid app is most needed; and audio of someone describing a
medical emergency never leaves the device.

Planned surface:

``transcribe(audio_path, *, language) -> Transcript``
    Invokes the whisper.cpp binary as a subprocess and parses its output.
    Called from the Celery ``ai.transcribe_audio`` task, never inline in a
    request -- transcription is measured in seconds.

Configuration: ``whisper_cpp_binary``, ``whisper_model_path``,
``whisper_language`` (default ``ne``), and ``whisper_threads``.
"""
