"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { AlertTriangle, ArrowUp, History, Phone, RotateCcw, Video } from "lucide-react";

import { BrandMark } from "@/components/brand";
import { ChatHistory } from "@/components/chat/chat-history";
import { ExampleExchange } from "@/components/chat/example-exchange";
import { MessageText } from "@/components/chat/message-text";
import { Button } from "@/components/ui/button";
import { SmoothInput } from "@/components/ui/smooth-input";
import { useCopy } from "@/lib/i18n";
import { useAuth } from "@/lib/store/auth";
import { useChatStore } from "@/lib/store/chat";
import { cn } from "@/lib/utils";

type Props = { onTalkToDoctor: () => void; className?: string };

function prefersReducedMotion(): boolean {
  return window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;
}

export function ChatPanel({ onTalkToDoctor, className }: Props) {
  const { messages, pending, error, urgent, send, retry, reset } = useChatStore();
  const { user, ready } = useAuth();
  const { t } = useCopy();
  const [showHistory, setShowHistory] = useState(false);
  const [draft, setDraft] = useState("");
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (messages.length === 0 && !pending) return;
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
      aria-label={t.chat.title}
      className={cn(
        "flex h-full min-h-0 flex-col overflow-hidden rounded-[12px] border bg-card",
        className,
      )}
    >
      <div className="flex items-center justify-between gap-3 border-b px-4 py-3 sm:px-5">
        <div className="flex min-w-0 items-center gap-2.5">
          <BrandMark className="size-8" />
          <div className="min-w-0">
            <h2 className="truncate text-[15px] font-semibold">{t.chat.title}</h2>
            <p className="truncate text-xs text-muted-foreground">{t.chat.subtitle}</p>
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
              {t.chat.myChats}
            </Button>
          )}
          {ready && !user && (
            <Link
              href="/account/?next=/patient/"
              className="inline-flex min-h-11 max-w-[6.5rem] items-center rounded-[8px] px-2 text-right text-xs font-medium leading-tight text-muted-foreground transition-colors duration-150 hover:text-foreground sm:min-h-0 sm:max-w-none sm:py-1"
            >
              {t.chat.signInToSave}
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
              {t.chat.newChat}
            </Button>
          )}
        </div>
      </div>

      {urgent && (
        <div
          role="alert"
          className="flex flex-col gap-3 border-b border-l-4 border-l-triage-red bg-triage-red/[0.06] px-4 py-3 sm:flex-row sm:items-center"
        >
          <p className="flex flex-1 items-start gap-2 text-sm font-semibold">
            <AlertTriangle aria-hidden className="mt-0.5 size-4 shrink-0 text-triage-red-text" />
            {t.chat.urgent}
          </p>
          <div className="flex gap-2">
            <Button asChild variant="emergency" size="sm">
              <a href="tel:102">
                <Phone aria-hidden />
                {t.chat.call102}
              </a>
            </Button>
            <Button variant="outline" size="sm" onClick={onTalkToDoctor}>
              {t.chat.talk}
            </Button>
          </div>
        </div>
      )}

      {showHistory && user ? (
        <div className="min-h-0 flex-1 overflow-y-auto py-2">
          <ChatHistory onOpened={() => setShowHistory(false)} />
        </div>
      ) : (
        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-5 sm:px-5">
          {messages.length === 0 && (
            <div className="flex flex-col gap-6">
              <div>
                <p className="font-display text-2xl">{t.chat.emptyTitle}</p>
                <p className="text-sm text-muted-foreground">{t.chat.emptyBody}</p>
              </div>
              <ExampleExchange />
              <div>
                <p className="mb-2 text-xs font-medium text-muted-foreground">{t.chat.tryLabel}</p>
                <div className="flex flex-wrap gap-2">
                  {t.chat.examples.map((example) => (
                    <button
                      key={example}
                      type="button"
                      onClick={() => submit(example)}
                      disabled={pending}
                      className="min-h-11 rounded-[8px] border bg-background px-3 py-2 text-left text-sm transition-colors duration-150 hover:border-foreground/30 hover:bg-accent disabled:opacity-50 sm:min-h-0 sm:py-1.5"
                    >
                      {example}
                    </button>
                  ))}
                </div>
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
                {message.role === "assistant" && <BrandMark className="mt-0.5 size-7" />}
                <div
                  className={cn(
                    "max-w-[85%] text-[0.95rem] leading-relaxed",
                    message.role === "user"
                      ? "rounded-[12px] rounded-br-[4px] bg-deep px-4 py-2.5 text-deep-foreground"
                      : "rounded-[12px] rounded-tl-[4px] bg-muted px-4 py-3 text-foreground",
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
              {t.chat.thinking}
            </div>
          )}

          {error && !pending && (
            <div
              role="alert"
              className="mt-4 flex flex-wrap items-center gap-3 rounded-[8px] border border-destructive/30 bg-destructive/[0.05] px-3.5 py-2.5 text-sm"
            >
              <span className="flex-1">{error}</span>
              <Button variant="outline" size="sm" onClick={retry}>
                {t.chat.tryAgain}
              </Button>
            </div>
          )}
          <div ref={endRef} />
        </div>
      )}

      <form
        className="border-t p-3 sm:px-4"
        onSubmit={(event) => {
          event.preventDefault();
          submit(draft);
        }}
      >
        <div className="flex items-center gap-2 rounded-[10px] border border-input bg-background p-1.5 pl-4 transition-colors duration-150 focus-within:border-ring">
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
            placeholder={t.chat.placeholder}
            aria-label={t.chat.message}
            wrapperClassName="flex-1 self-center"
            className="h-11 text-base sm:h-9"
          />
          <Button
            type="submit"
            size="icon"
            className="size-11 rounded-[6px] sm:size-9"
            disabled={pending || !draft.trim()}
            aria-label={t.chat.send}
          >
            <ArrowUp aria-hidden />
          </Button>
        </div>
        <Button
          type="button"
          variant="outline"
          data-testid="talk-to-professional"
          className="mt-2 h-11 w-full"
          onClick={onTalkToDoctor}
        >
          <Video aria-hidden />
          {t.chat.talk}
        </Button>
      </form>
    </section>
  );
}
