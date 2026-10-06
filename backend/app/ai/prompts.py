"""System prompt, history trimming, and the urgent-symptom check."""

from __future__ import annotations

import re

from app.schemas.chat import ChatMessage

# The answer to a request that is plainly not about health (see
# is_off_topic). Fixed text: the request never reaches the model, so it gets
# nothing to improvise on.
OFF_TOPIC_REPLY_EN = (
    "I can only help with health and first-aid questions. "
    "If someone is in danger, call 102."
)
OFF_TOPIC_REPLY_NE = (
    "म स्वास्थ्य र प्राथमिक उपचारसम्बन्धी प्रश्नमा मात्र सहयोग गर्न सक्छु। "
    "कसैको ज्यान जोखिममा छ भने 102 मा फोन गर्नुहोस्।"
)

# Short on purpose. A longer list of rules made the 4B model start planning
# out loud ("I should ... Plan: 1. Acknowledge") instead of answering.
SYSTEM_PROMPT = """\
You are SwasthyaSetu, a first-aid assistant for people in Nepal who may be far \
from medical help.

Rules:
- Only answer questions about health, illness, injuries and first aid. For \
anything else, say in one sentence that you can only help with health and \
first-aid questions.
- Reply in the same language the user writes in. If they write in Nepali, \
reply in Nepali (Devanagari script). Otherwise reply in English.
- Start straight away with the advice. Do not introduce yourself or repeat \
the question back.
- Give practical first-aid steps as a short numbered list. Keep the whole \
reply under 180 words.
- You do not diagnose. Say what the symptoms *may* suggest and what to do next.
- Never give medicine doses.
- If anything sounds life-threatening (chest pain, trouble breathing, heavy \
bleeding, unconsciousness, seizures, stroke signs, severe burns, poisoning), \
start your reply by telling them to call 102 for an ambulance and to press \
"Talk to a professional" now.
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


# MedGemma 1.5 can "think" before answering: <unused94>thought ... <unused95>,
# then the reply. The reasoning is not for patients.
_THOUGHT_START = "<unused94>"
_THOUGHT_END = "<unused95>"


def strip_thinking(text: str) -> str:
    """Return only the patient-facing reply. Empty if the model never got past
    its reasoning (e.g. it ran out of tokens mid-thought)."""
    if _THOUGHT_END in text:
        text = text.rsplit(_THOUGHT_END, 1)[1]
    elif text.lstrip().startswith(_THOUGHT_START):
        return ""
    # A thought block whose markers were lost in decoding starts with the bare
    # word "thought". There is no telling where its reasoning ends, so none
    # of it is shown.
    if _BARE_THOUGHT_RE.match(text):
        return ""
    return strip_preamble(text.strip())


_BARE_THOUGHT_RE = re.compile(r"\s*thought\s*\n", re.IGNORECASE)


# With its thinking step skipped, MedGemma sometimes plans out loud in the
# reply instead: "Okay, I understand. I need to provide first-aid advice ...
# in English, using a numbered list, under 180 words ...". That paragraph
# restates our instructions; the patient should only ever see the advice.
_PREAMBLE_OPENERS = re.compile(
    r"^\s*(okay|ok|alright|sure|understood|got it|i understand|ठीक छ)\b",
    re.IGNORECASE,
)
_PREAMBLE_MARKERS = re.compile(
    r"\b(i need to|i should|i must|i will|i'll|i am going to|i'm going to|"
    r"let me|the user|my (task|instructions|response)|numbered list|"
    r"\d+ words|medicine doses|in (english|nepali)|starting directly)\b",
    re.IGNORECASE,
)


def _is_preamble(paragraph: str) -> bool:
    if _PREAMBLE_OPENERS.search(paragraph):
        return True
    return len(_PREAMBLE_MARKERS.findall(paragraph)) >= 2


def strip_preamble(text: str) -> str:
    """Drop leading paragraphs in which the model talks about its task rather
    than to the patient. Never empties a reply: a lone paragraph stays."""
    paragraphs = re.split(r"\n\s*\n", text.strip())
    while len(paragraphs) > 1 and _is_preamble(paragraphs[0]):
        paragraphs.pop(0)
    return "\n\n".join(paragraphs).strip()


_DEVANAGARI_RE = re.compile(r"[\u0900-\u097F]")  # Devanagari block

# Appended to the latest user turn only in what the model sees. A small model
# follows a reminder at the end of the prompt far more reliably than the
# language rule in the system prompt. Kept to the language alone: a longer
# reminder gets recited back as a preamble.
NEPALI_REMINDER = "(नेपालीमा, देवनागरी लिपिमा जवाफ दिनुहोस्। Reply in Nepali.)"
ENGLISH_REMINDER = "(Reply in English.)"


def is_nepali(text: str) -> bool:
    return bool(_DEVANAGARI_RE.search(text))


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

    if merged and merged[-1]["role"] == "user":
        last = merged[-1]["content"]
        reminder = NEPALI_REMINDER if is_nepali(last) else ENGLISH_REMINDER
        merged[-1] = {"role": "user", "content": last + "\n\n" + reminder}

    return [{"role": "system", "content": SYSTEM_PROMPT}, *merged]


# --- Off-topic requests ------------------------------------------------------
# Plainly non-health requests (code, homework, stories, attempts to change
# the rules) are answered with OFF_TOPIC_REPLY_* without calling the model.
# Deliberately narrow: anything that also mentions a symptom, an injury or
# care, and anything is_urgent() flags, always goes to the model, so this can
# never stand between someone and first aid.
_OFF_TOPIC_RE = re.compile(
    "|".join(
        [
            r"\b(java|javascript|typescript|python|c\+\+|c#|php|sql|html|css|"
            r"react|node\.?js|kotlin|rust|golang)\b",
            r"\b(code|coding|programm?ing|program|algorithm|polymorphism|"
            r"inheritance|compiler?|debug\w*|website|software)\b",
            r"\b(poem|poetry|story|stories|essay|song|lyrics|joke|riddle|novel)\b",
            r"\b(homework|assignment|maths?|mathematics|equation|calculus|algebra)\b",
            r"\b(translate|translation)\b",
            r"\b(bitcoin|crypto\w*|stock market|forex|betting|lottery)\b",
            r"\b(cricket|football|movie|film|netflix|celebrity|election|"
            r"politic\w*)\b",
            r"\b(hack|hacking|password|phishing)\b",
            r"ignore (all |your |the |previous |above )*(instructions|rules|prompt)",
            r"\b(system prompt|jailbreak|role-?play|pretend to be|act as)\b",
            "कविता",  # poem
            "कथा",  # story
            "गृहकार्य",  # homework
            "गणित",  # maths
            "चुटकिला",  # joke
        ]
    ),
    re.IGNORECASE,
)
_HEALTH_RE = re.compile(
    "|".join(
        [
            r"\b(\w*ache\w*|pain\w*|hurt\w*|fever|bleed\w*|blood|burn\w*|cut|"
            r"wound\w*|injur\w*|fell|fall\w*|broke\w*|fracture\w*|swell\w*|"
            r"swollen|sick|ill|illness|vomit\w*|diarrh\w*|cough\w*|breath\w*|"
            r"chest|head|stomach|pregnan\w*|baby|child|medicine\w*|medication\w*|"
            r"tablet\w*|pill\w*|dose|doctor\w*|hospital|clinic|nurse\w*|"
            r"poison\w*|bite|bitten|allerg\w*|rash|dizz\w*|faint\w*|"
            r"unconscious|seizure\w*|anxiety|anxious|depress\w*|panic|"
            r"health\w*|first aid|swallow\w*|chok\w*|drown\w*|snake|sting\w*|"
            r"symptom\w*|disease|infection|eye|ear|tooth|teeth)\b",
            "दुख",  # pain, hurting
            "ज्वरो",  # fever
            "रगत",  # blood
            "पोल",  # burn
            "चोट",  # injury
            "बिरामी",  # ill, patient
            "औषधि",  # medicine
            "डाक्टर",  # doctor
            "बच्चा",  # child
            "सुन्नि",  # swelling
        ]
    ),
    re.IGNORECASE,
)


def is_off_topic(text: str) -> bool:
    if is_urgent(text) or _HEALTH_RE.search(text):
        return False
    return bool(_OFF_TOPIC_RE.search(text))


def off_topic_reply(text: str) -> str:
    return OFF_TOPIC_REPLY_NE if is_nepali(text) else OFF_TOPIC_REPLY_EN
