"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import {
  AlertTriangle,
  ArrowUp,
  History,
  Phone,
  RotateCcw,
  Stethoscope,
} from "lucide-react";

import { BrandMark } from "@/components/brand";
import { ChatHistory } from "@/components/chat/chat-history";
import { MessageText } from "@/components/chat/message-text";
import { Button } from "@/components/ui/button";
import { SmoothInput } from "@/components/ui/smooth-input";
import { useAuth } from "@/lib/store/auth";
import { useChatStore } from "@/lib/store/chat";
import { cn } from "@/lib/utils";

const EXAMPLES = [
  "I burned my hand while cooking",
  "मेरो बच्चालाई २ दिनदेखि ज्वरो आएको छ", // My child has had a fever for 2 days
  "Someone fell and their ankle is swollen",
];

type Props = { onTalkToDoctor: () => void; className?: string };

function prefersReducedMotion(): boolean {
  return window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;
}

export function ChatPanel({ onTalkToDoctor, className }: Props) {
  const { messages, pending, error, urgent, send, retry, reset } = useChatStore();
  const { user, ready } = useAuth();
  const [showHistory, setShowHistory] = useState(false);
  const [draft, setDraft] = useState("");
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({
      behavior: prefersReducedMotion() ? "auto" : "smooth",
      block: "end",
    });
  }, [messages.length, pending]);

  const submit = async (text: string) => {
    if (!text.trim() || pending) return;
    setDraft("");
    await send(text);
  };

  return (
    <section
      aria-label="First-aid assistant"
      className={cn(
        "flex h-full min-h-0 flex-col overflow-hidden rounded-[20px] border bg-card shadow-soft",
        className,
      )}
    >
      <div className="flex items-center justify-between gap-3 border-b px-4 py-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-muted shadow-raised-sm">
            <BrandMark className="size-6" />
          </span>
          <div className="min-w-0">
            <h2 className="truncate text-[15px] font-bold">First-aid assistant</h2>
            <p className="truncate text-xs text-muted-foreground">
              English or नेपाली · Not a diagnosis
            </p>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          {user && (
            <Button
              variant={showHistory ? "secondary" : "ghost"}
              size="sm"
              onClick={() => setShowHistory((open) => !open)}
              aria-pressed={showHistory}
            >
              <History aria-hidden />
              My chats
            </Button>
          )}
          {ready && !user && (
            <Link
              href="/account/"
              className="inline-flex min-h-11 items-center rounded-full px-2.5 text-xs font-bold text-muted-foreground transition-colors duration-200 hover:text-primary-text sm:min-h-0 sm:py-1"
            >
              Sign in to save chats
            </Link>
          )}
          {(messages.length > 0 || showHistory) && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                reset();
                setShowHistory(false);
              }}
              disabled={pending}
            >
              <RotateCcw aria-hidden />
              New chat
            </Button>
          )}
        </div>
      </div>

      {urgent && (
        <div
          role="alert"
          className="flex flex-col gap-3 border-b border-l-4 border-l-triage-red bg-triage-red/[0.07] px-4 py-3 sm:flex-row sm:items-center"
        >
          <p className="flex flex-1 items-start gap-2 text-sm font-semibold">
            <AlertTriangle aria-hidden className="mt-0.5 size-4 shrink-0 text-triage-red" />
            This may be an emergency. Call 102 now, or talk to a doctor immediately.
          </p>
          <div className="flex gap-2">
            <Button asChild variant="emergency" size="sm">
              <a href="tel:102">
                <Phone aria-hidden />
                Call 102
              </a>
            </Button>
            <Button variant="outline" size="sm" onClick={onTalkToDoctor}>
              Talk to a doctor
            </Button>
          </div>
        </div>
      )}

      {showHistory && user ? (
        <div className="min-h-0 flex-1 overflow-y-auto py-2">
          <ChatHistory onOpened={() => setShowHistory(false)} />
        </div>
      ) : (
        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-5">
          {messages.length === 0 && (
            <div className="flex h-full flex-col justify-center gap-5 py-4">
              <div>
                <p className="text-lg font-bold tracking-tight">What is happening?</p>
                <p className="text-sm text-muted-foreground">
                  Describe it in your own words. के भइरहेको छ, लेख्नुहोस्।
                </p>
              </div>
              <div className="flex flex-col items-start gap-2">
                {EXAMPLES.map((example) => (
                  <button
                    key={example}
                    type="button"
                    onClick={() => submit(example)}
                    disabled={pending}
                    className="min-h-11 rounded-full border bg-card px-4 py-2 text-left text-sm shadow-raised-sm transition-[background-color,box-shadow] duration-200 hover:bg-accent active:shadow-pressed disabled:opacity-50 sm:min-h-0 sm:py-1.5"
                  >
                    {example}
                  </button>
                ))}
              </div>
            </div>
          )}

          <ol aria-live="polite" aria-label="Conversation" className="flex flex-col gap-4">
            {messages.map((message, i) => (
              <li
                key={`${i}-${message.role}`}
                data-role={message.role}
                className={cn(
                  "flex gap-2.5",
                  message.role === "user" ? "justify-end" : "justify-start",
                )}
              >
                {message.role === "assistant" && (
                  <span
                    aria-hidden
                    className="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-full bg-muted"
                  >
                    <BrandMark className="size-[18px]" />
                  </span>
                )}
                <div
                  className={cn(
                    "max-w-[85%] text-[0.95rem] leading-relaxed",
                    message.role === "user"
                      ? "rounded-[20px] rounded-br-md bg-primary px-4 py-2.5 text-primary-foreground shadow-raised-sm"
                      : "rounded-[20px] rounded-tl-md bg-muted px-4 py-3 text-foreground",
                  )}
                >
                  <MessageText text={message.content} />
                </div>
              </li>
            ))}
          </ol>

          {pending && (
            <div role="status" className="mt-4 flex items-center gap-2.5 text-sm text-muted-foreground">
              <span aria-hidden className="flex gap-1">
                <span className="size-1.5 animate-pulse rounded-full bg-muted-foreground" />
                <span className="size-1.5 animate-pulse rounded-full bg-muted-foreground [animation-delay:150ms]" />
                <span className="size-1.5 animate-pulse rounded-full bg-muted-foreground [animation-delay:300ms]" />
              </span>
              Thinking… the first reply can take up to a minute while the model wakes up.
            </div>
          )}

          {error && !pending && (
            <div
              role="alert"
              className="mt-4 flex flex-wrap items-center gap-3 rounded-[14px] border border-destructive/30 bg-destructive/[0.05] px-3.5 py-2.5 text-sm"
            >
              <span className="flex-1">{error}</span>
              <Button variant="outline" size="sm" onClick={retry}>
                Try again
              </Button>
            </div>
          )}
          <div ref={endRef} />
        </div>
      )}

      <form
        className="border-t p-3"
        onSubmit={(event) => {
          event.preventDefault();
          submit(draft);
        }}
      >
        <div className="flex items-center gap-2 rounded-full border border-input bg-background p-1.5 pl-5 shadow-pressed transition-colors duration-200 focus-within:border-ring">
          <SmoothInput
            name="message"
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={(event) => {
              // Enter while a Nepali (or other) input method is still
              // composing confirms the word; it must not send the message.
              if (event.key === "Enter" && event.nativeEvent.isComposing) {
                event.preventDefault();
              }
            }}
            maxLength={2000}
            autoComplete="off"
            enterKeyHint="send"
            placeholder="Describe the symptoms… / लक्षण लेख्नुहोस्…"
            aria-label="Message"
            wrapperClassName="flex-1 self-center"
            className="h-11 text-base sm:h-9"
          />
          <Button
            type="submit"
            size="icon"
            className="size-11 sm:size-9"
            disabled={pending || !draft.trim()}
            aria-label="Send"
          >
            <ArrowUp aria-hidden />
          </Button>
        </div>
      </form>

      <div className="px-3 pb-3">
        <Button variant="cta" className="h-11 w-full" onClick={onTalkToDoctor}>
          <Stethoscope aria-hidden />
          Talk to a doctor · डाक्टरसँग कुरा गर्नुहोस्
        </Button>
      </div>
    </section>
  );
}
