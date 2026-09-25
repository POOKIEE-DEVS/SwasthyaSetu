"""SwasthyaSetu: MedGemma inference Space.

Deploy this folder as a Hugging Face Space (Gradio SDK, GPU hardware). It
exposes:

- ``/generate`` API: the backend sends the conversation as a JSON string and
  gets the reply text back.
- A chat UI at the Space URL, for testing the model directly.

Environment (Space settings > Variables and secrets):
- ``HF_TOKEN`` (secret): a token whose account has accepted the MedGemma terms.
- ``MODEL_ID`` (optional): defaults to google/medgemma-1.5-4b-it.
- ``MAX_NEW_TOKENS`` (optional): defaults to 400.
- ``SWASTHYA_MOCK_MODEL=1``: skip the model entirely (local protocol testing).

Works on dedicated GPU hardware (A10G / L4) and on ZeroGPU.
"""

from __future__ import annotations

import json
import os

# On ZeroGPU, `spaces` must be imported before torch. Elsewhere its decorator
# is a no-op, and off Hugging Face it is not installed at all.
try:
    import spaces

    gpu = spaces.GPU(duration=90)
except ImportError:

    def gpu(fn):
        return fn


import gradio as gr

MODEL_ID = os.environ.get("MODEL_ID", "google/medgemma-1.5-4b-it")
MAX_NEW_TOKENS = int(os.environ.get("MAX_NEW_TOKENS", "400"))
MAX_INPUT_TOKENS = int(os.environ.get("MAX_INPUT_TOKENS", "3000"))
MOCK = os.environ.get("SWASTHYA_MOCK_MODEL") == "1"

UI_SYSTEM_PROMPT = (
    "You are a first-aid assistant for people in Nepal. Reply in the user's "
    "language (English or Nepali). Give short, practical first-aid steps. Do "
    "not diagnose and never give medicine doses. For anything life-threatening, "
    "tell them to call 102 for an ambulance first."
)

model = None
processor = None
torch = None

if not MOCK:
    import torch
    from transformers import AutoModelForImageTextToText, AutoProcessor

    token = os.environ.get("HF_TOKEN")
    processor = AutoProcessor.from_pretrained(MODEL_ID, token=token)
    model = AutoModelForImageTextToText.from_pretrained(
        MODEL_ID, dtype=torch.bfloat16, token=token
    )
    # On ZeroGPU, moving to "cuda" at import time is the documented pattern;
    # the GPU is attached when a @spaces.GPU function runs.
    use_cuda = torch.cuda.is_available() or bool(os.environ.get("SPACES_ZERO_GPU"))
    model.to("cuda" if use_cuda else "cpu")
    model.eval()


def normalise(raw: list) -> list[dict]:
    """Convert [{role, content: str}] into the processor's format: optional
    system turn first, then strictly alternating user/assistant turns
    starting with a user turn.
    """
    turns: list[dict] = []
    system_text = None
    for item in raw:
        if not isinstance(item, dict):
            continue
        role = item.get("role")
        text = str(item.get("content", "")).strip()
        if not text:
            continue
        if role == "system":
            system_text = text
            continue
        if role not in ("user", "assistant"):
            continue
        if turns and turns[-1]["role"] == role:
            turns[-1]["text"] += "\n\n" + text
        else:
            turns.append({"role": role, "text": text})

    while turns and turns[0]["role"] != "user":
        turns.pop(0)

    messages = []
    if system_text:
        messages.append(
            {"role": "system", "content": [{"type": "text", "text": system_text}]}
        )
    messages += [
        {"role": t["role"], "content": [{"type": "text", "text": t["text"]}]}
        for t in turns
    ]
    return messages


def fit_to_context(messages: list[dict]) -> list[dict]:
    """Drop the oldest exchanges until the prompt fits MAX_INPUT_TOKENS."""
    has_system = bool(messages) and messages[0]["role"] == "system"
    head = messages[:1] if has_system else []
    body = messages[1:] if has_system else messages[:]

    while len(body) > 1:
        encoded = processor.apply_chat_template(
            head + body,
            add_generation_prompt=True,
            tokenize=True,
            return_dict=True,
            return_tensors="pt",
        )
        # Tensor shape is unambiguous; a plain list here may be batched
        # ([[ids]]), whose len() is always 1.
        if encoded["input_ids"].shape[-1] <= MAX_INPUT_TOKENS:
            break
        body = body[2:] if len(body) > 2 else body[-1:]
    return head + body


@gpu
def run_model(messages: list[dict]) -> str:
    inputs = processor.apply_chat_template(
        messages,
        add_generation_prompt=True,
        tokenize=True,
        return_dict=True,
        return_tensors="pt",
    ).to(model.device, dtype=torch.bfloat16)
    input_len = inputs["input_ids"].shape[-1]
    with torch.inference_mode():
        output = model.generate(
            **inputs, max_new_tokens=MAX_NEW_TOKENS, do_sample=False
        )
    return processor.decode(output[0][input_len:], skip_special_tokens=True).strip()


def reply_to(messages: list[dict]) -> str:
    if not messages or messages[-1]["role"] != "user":
        raise gr.Error("The conversation must end with a user message.")
    if MOCK:
        last = messages[-1]["content"][0]["text"]
        return f"[mock MedGemma] You said: {last}"
    return run_model(fit_to_context(messages))


def generate(messages_json: str) -> str:
    """API endpoint used by the SwasthyaSetu backend."""
    if len(messages_json) > 100_000:
        raise gr.Error("Request too large.")
    try:
        raw = json.loads(messages_json)
    except json.JSONDecodeError as exc:
        raise gr.Error("messages_json must be a JSON list of messages.") from exc
    if not isinstance(raw, list):
        raise gr.Error("messages_json must be a JSON list of messages.")
    return reply_to(normalise(raw))


def chat_ui(message: str, history: list[dict]) -> str:
    """Chat tab on the Space page, for testing the model by hand."""
    raw = [{"role": "system", "content": UI_SYSTEM_PROMPT}]
    for turn in history:
        content = turn.get("content")
        if isinstance(content, str):
            raw.append({"role": turn.get("role"), "content": content})
    raw.append({"role": "user", "content": message})
    return reply_to(normalise(raw))


with gr.Blocks(title="SwasthyaSetu · MedGemma") as demo:
    gr.Markdown(
        f"# SwasthyaSetu · MedGemma\n"
        f"Model: `{MODEL_ID}`{' (MOCK MODE)' if MOCK else ''}. "
        "First-aid information only, not a diagnosis."
    )
    gr.ChatInterface(fn=chat_ui)
    gr.api(generate, api_name="generate")

# One generation at a time: the GPU can't usefully run two at once, and
# queued requests are served in order.
demo.queue(default_concurrency_limit=1)

if __name__ == "__main__":
    demo.launch()
