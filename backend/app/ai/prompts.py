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

# The instruction the model gets, as a few plain sentences. Written as a
# rules list, or with a reminder appended to the patient's message, the 4B
# model treated it as a task to analyse ("The user has asked ... Identify the
# core question ... Plan: ...") and that analysis became the reply. Plain
# prose, like the prompt on the Space's own test page, gets plain answers.
SYSTEM_PROMPT_EN = (
    "You are SwasthyaSetu, a first-aid assistant for people in Nepal. Reply in "
    "English. Give short, practical first-aid steps as a numbered list, under "
    "180 words. Do not diagnose and never give medicine doses. For anything "
    "life-threatening, first tell them to call 102 for an ambulance and to "
    'press "Talk to a professional". You only help with health, illness, '
    "injuries and first aid; for anything else, say in one sentence that you "
    "can only help with health and first-aid questions."
)

# For a message in Nepali the instruction itself is in Nepali: that steers
# the reply into Nepali without asking the model to "translate", which it
# then plans out loud.
SYSTEM_PROMPT_NE = (
    "तपाईं स्वास्थ्य सेतु हुनुहुन्छ, नेपालका मानिसहरूका लागि प्राथमिक उपचार "
    "सहायक। सधैं नेपाली भाषामा, देवनागरी लिपिमा जवाफ दिनुहोस्। अहिले के गर्ने "
    "भन्ने छोटा, व्यावहारिक प्राथमिक उपचारका कदमहरू नम्बर लगाएर दिनुहोस्। रोग "
    "निदान नगर्नुहोस् र औषधिको मात्रा कहिल्यै नबताउनुहोस्। ज्यान जोखिममा देखिए "
    'सबैभन्दा पहिले 102 मा एम्बुलेन्स बोलाउन र "स्वास्थ्यकर्मीसँग कुरा '
    'गर्नुहोस्" थिच्न भन्नुहोस्। स्वास्थ्यसँग सम्बन्धित नभएका प्रश्नमा, तपाईं '
    "स्वास्थ्य र प्राथमिक उपचारमा मात्र सहयोग गर्न सक्नुहुन्छ भनेर एक वाक्यमा "
    "भन्नुहोस्।"
)

# Kept for the tests and anything that wants "the" prompt.
SYSTEM_PROMPT = SYSTEM_PROMPT_EN

# Word for word the prompt of the Space's test page, which answers well. Used
# for one retry when a reply still comes back as the model's own analysis.
FALLBACK_SYSTEM_PROMPT = (
    "You are a first-aid assistant for people in Nepal. Reply in the user's "
    "language (English or Nepali). Give short, practical first-aid steps. Do "
    "not diagnose and never give medicine doses. For anything life-threatening, "
    "tell them to call 102 for an ambulance first."
)

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

def is_nepali(text: str) -> bool:
    return bool(_DEVANAGARI_RE.search(text))


def is_urgent(text: str) -> bool:
    return bool(_URGENT_RE.search(text))


def build_model_messages(
    history: list[ChatMessage],
    *,
    max_messages: int,
    system_prompt: str | None = None,
) -> list[dict[str, str]]:
    """System prompt plus the most recent turns, in a shape the Gemma chat
    template accepts: starting with a user turn and strictly alternating.
    The prompt follows the language of the latest message unless given.
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

    if system_prompt is None:
        latest = merged[-1]["content"] if merged else ""
        system_prompt = SYSTEM_PROMPT_NE if is_nepali(latest) else SYSTEM_PROMPT_EN
    # The patient's words go to the model exactly as written.
    return [{"role": "system", "content": system_prompt}, *merged]


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


# --- Leaked analysis -----------------------------------------------------------
# Sometimes the model answers with its working instead of the answer: "The
# user has asked ... 1. Identify the core question 2. Assess urgency 3.
# Formulate advice 4. Translate to Nepali". Such a reply is never shown; the
# chat endpoint retries once with FALLBACK_SYSTEM_PROMPT.
_REASONING_RE = re.compile(
    "|".join(
        [
            r"\bthe user(?:'s)? (?:has )?(?:asked|request|is asking|wants|"
            r"question|message|language|writes)\b",
            r"\bidentify (?:the )?(?:core |main )?(?:question|issue|problem)\b",
            r"\bassess (?:the )?(?:urgency|situation|severity)\b",
            r"\bformulate (?:the )?(?:advice|response|answer|reply)\b",
            r"\btranslate (?:it |this |the advice )?(?:in)?to (?:nepali|english)\b",
            r"^\s*\**plan:?\**\s*$",
            r"\bI should (?:start|begin|reply|respond|provide|tell|mention)\b",
            r"\bmy (?:response|reply|answer) (?:should|will|must)\b",
        ]
    ),
    re.IGNORECASE | re.MULTILINE,
)


def looks_like_reasoning(reply: str) -> bool:
    """True when a reply reads as the model's own analysis, not advice."""
    return bool(_REASONING_RE.search(reply))
