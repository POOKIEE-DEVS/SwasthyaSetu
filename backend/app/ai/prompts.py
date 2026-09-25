"""System prompt, history trimming, and the urgent-symptom check."""

from __future__ import annotations

import re

from app.schemas.chat import ChatMessage

SYSTEM_PROMPT = """\
You are SwasthyaSetu, a first-aid assistant for people in Nepal who may be far \
from medical help.

Rules:
- Reply in the same language the user writes in. If they write in Nepali, \
reply in Nepali (Devanagari script). Otherwise reply in English.
- Give practical first-aid steps as a short numbered list. Keep the whole \
reply under 180 words.
- You do not diagnose. Say what the symptoms *may* suggest and what to do next.
- Never give medicine doses.
- If anything sounds life-threatening (chest pain, trouble breathing, heavy \
bleeding, unconsciousness, seizures, stroke signs, severe burns, poisoning), \
start your reply by telling them to call 102 for an ambulance and to press \
"Talk to a doctor" now.
- For anything that is not getting better, recommend talking to a doctor.
"""

# Keyword check for obvious emergencies. It is deliberately crude and
# conservative: its only job is to surface the "call 102 / talk to a doctor"
# banner immediately, without waiting on the model. It is not triage.
_URGENT_PATTERNS = [
    r"chest pain",
    r"heart attack",
    r"can'?t breathe",
    r"cannot breathe",
    r"not breathing",
    r"difficulty breathing",
    r"unconscious",
    r"fainted",
    r"seizure",
    r"\bfits\b",
    r"heavy bleeding",
    r"bleeding a lot",
    r"stroke",
    r"poison",
    r"snake ?bite",
    r"severe burn",
    r"suicid",
    "छाती दुख",  # chest pain
    "सास फेर्न",  # breathing (difficulty)
    "बेहोस",  # unconscious
    "धेरै रगत",  # a lot of blood
    "रगत बगि",  # bleeding
    "छारे",  # seizure
    "सर्पले टोक",  # snake bite
    "विष खा",  # ate poison (bare विष would match विषय, "topic")
]
_URGENT_RE = re.compile("|".join(_URGENT_PATTERNS), re.IGNORECASE)


def is_urgent(text: str) -> bool:
    return bool(_URGENT_RE.search(text))


def build_model_messages(
    history: list[ChatMessage], *, max_messages: int
) -> list[dict[str, str]]:
    """System prompt plus the most recent turns, in a shape the Gemma chat
    template accepts: starting with a user turn and strictly alternating.
    """
    recent = history[-max_messages:]

    # Drop leading assistant turns left over from trimming.
    while recent and recent[0].role != "user":
        recent = recent[1:]

    # Merge consecutive same-role turns (e.g. a retried user message).
    merged: list[dict[str, str]] = []
    for message in recent:
        if merged and merged[-1]["role"] == message.role:
            merged[-1]["content"] += "\n\n" + message.content
        else:
            merged.append({"role": message.role, "content": message.content})

    return [{"role": "system", "content": SYSTEM_PROMPT}, *merged]
