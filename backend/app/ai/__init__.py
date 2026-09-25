"""AI & Edge ML engine.

Three models, each with a distinct job and a distinct place to run:

``llm``
    MedGemma -- the clinical triage model. Reached over HTTP, so it can be
    an Ollama container in development and a hosted endpoint in production
    without any code change.
``stt``
    whisper.cpp -- Nepali speech-to-text. Runs on-device as a subprocess.
    Voice input is the point: a caregiver in an emergency should be able to
    describe what they see out loud, and many users are more fluent speaking
    Nepali than typing it.
``tts``
    Localized Nepali narration, so first-aid steps can be *heard* while both
    hands are busy with the patient -- and read aloud to users who cannot
    read comfortably.

The ``triage`` subpackage orchestrates these into the six-stage pipeline of
architecture section 3. Everything here is invoked from Celery tasks rather
than directly from a request handler: inference takes seconds, and an
emergency request must never be blocked behind someone else's model call.
"""
