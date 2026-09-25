"use client";

import { useEffect, useRef, useState } from "react";
import { AlertTriangle, Loader2, RotateCcw, SendHorizontal, Stethoscope } from "lucide-react";

import { MessageText } from "@/components/chat/message-text";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { useChatStore } from "@/lib/store/chat";
import { cn } from "@/lib/utils";

const EXAMPLES = [
  "I burned my hand while cooking",
  "मेरो बच्चालाई २ दिनदेखि ज्वरो आएको छ", // My child has had a fever for 2 days
  "Someone fell and their ankle is swollen",
];

type Props = { onTalkToDoctor: () => void };

export function ChatPanel({ onTalkToDoctor }: Props) {
  const { messages, pending, error, urgent, send, retry, reset } = useChatStore();
  const [draft, setDraft] = useState("");
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [messages.length, pending]);

  const submit = async (text: string) => {
    if (!text.trim() || pending) return;
    setDraft("");
    await send(text);
  };

  return (
    <div className="flex h-full min-h-0 flex-col rounded-xl border bg-card">
      <div className="flex items-center justify-between border-b px-4 py-3">
        <div>
          <h2 className="font-semibold">First-aid assistant</h2>
          <p className="text-xs text-muted-foreground">English or नेपाली · Not a diagnosis</p>
        </div>
        {messages.length > 0 && (
          <Button variant="ghost" size="sm" onClick={reset} disabled={pending}>
            <RotateCcw aria-hidden />
            New chat
          </Button>
        )}
      </div>

      {urgent && (
        <div
          role="alert"
          className="flex flex-col gap-3 border-b bg-triage-red/10 px-4 py-3 sm:flex-row sm:items-center"
        >
          <p className="flex flex-1 items-start gap-2 text-sm font-medium">
            <AlertTriangle aria-hidden className="mt-0.5 size-4 shrink-0 text-triage-red" />
            This may be an emergency. Call 102 now, or talk to a doctor immediately.
          </p>
          <div className="flex gap-2">
            <Button asChild variant="emergency" size="sm">
              <a href="tel:102">Call 102</a>
            </Button>
            <Button variant="outline" size="sm" onClick={onTalkToDoctor}>
              Talk to a doctor
            </Button>
          </div>
        </div>
      )}

      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-4 py-4" aria-live="polite">
        {messages.length === 0 && (
          <div className="flex flex-col gap-3 py-6 text-center">
            <p className="text-muted-foreground">
              Describe what is happening, in English or Nepali.
              <br />
              <span className="text-sm">के भइरहेको छ, लेख्नुहोस्।</span>
            </p>
            <div className="flex flex-wrap justify-center gap-2">
              {EXAMPLES.map((example) => (
                <button
                  key={example}
                  type="button"
                  onClick={() => submit(example)}
                  className="rounded-full border px-3 py-1.5 text-sm hover:bg-accent"
                >
                  {example}
                </button>
              ))}
            </div>
          </div>
        )}

        {messages.map((message, i) => (
          <div
            key={i}
            data-role={message.role}
            className={cn("flex", message.role === "user" ? "justify-end" : "justify-start")}
          >
            <div
              className={cn(
                "max-w-[85%] rounded-2xl px-4 py-2.5 text-[0.95rem]",
                message.role === "user"
                  ? "rounded-br-sm bg-primary text-primary-foreground"
                  : "rounded-bl-sm bg-secondary text-secondary-foreground",
              )}
            >
              <MessageText text={message.content} />
            </div>
          </div>
        ))}

        {pending && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 aria-hidden className="size-4 animate-spin" />
            Thinking… the first reply can take up to a minute while the model wakes up.
          </div>
        )}

        {error && !pending && (
          <div role="alert" className="flex flex-wrap items-center gap-3 rounded-lg border border-destructive/40 bg-destructive/5 px-3 py-2 text-sm">
            <span className="flex-1">{error}</span>
            <Button variant="outline" size="sm" onClick={retry}>
              Try again
            </Button>
          </div>
        )}
        <div ref={endRef} />
      </div>

      <form
        className="flex items-end gap-2 border-t p-3"
        onSubmit={(event) => {
          event.preventDefault();
          submit(draft);
        }}
      >
        <Textarea
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter" && !event.shiftKey && !event.nativeEvent.isComposing) {
              event.preventDefault();
              submit(draft);
            }
          }}
          rows={1}
          maxLength={2000}
          placeholder="Describe the symptoms… / लक्षण लेख्नुहोस्…"
          aria-label="Message"
          className="max-h-40"
        />
        <Button type="submit" size="icon" disabled={pending || !draft.trim()} aria-label="Send">
          <SendHorizontal aria-hidden />
        </Button>
      </form>

      <div className="border-t px-3 py-2">
        <Button variant="outline" className="w-full" onClick={onTalkToDoctor}>
          <Stethoscope aria-hidden />
          Talk to a doctor · डाक्टरसँग कुरा गर्नुहोस्
        </Button>
      </div>
    </div>
  );
}
