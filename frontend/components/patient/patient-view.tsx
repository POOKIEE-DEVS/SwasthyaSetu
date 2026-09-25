"use client";

import { useEffect, useState } from "react";
import { ArrowLeft, Loader2, Stethoscope } from "lucide-react";

import { VideoCall } from "@/components/call/video-call";
import { ChatPanel } from "@/components/chat/chat-panel";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { api, ApiError } from "@/lib/api";
import { clearTicket, saveTicket, useTicket } from "@/lib/session";
import { transcript, useChatStore } from "@/lib/store/chat";

type Stage = "chat" | "request";

export function PatientView() {
  const messages = useChatStore((s) => s.messages);
  const [stage, setStage] = useState<Stage>("chat");
  // An in-progress call (restored after a refresh) takes over the page.
  const ticket = useTicket("patient");
  const [name, setName] = useState("");
  const [share, setShare] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Restore the chat after a refresh (the store skips automatic hydration).
  useEffect(() => {
    useChatStore.persist.rehydrate();
  }, []);

  const requestDoctor = async () => {
    if (!name.trim()) return;
    setSubmitting(true);
    setError(null);
    try {
      const summary = share ? transcript(messages) : null;
      const newTicket = await api.requestDoctor(name.trim(), summary);
      saveTicket(newTicket);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Couldn't request a doctor.");
    } finally {
      setSubmitting(false);
    }
  };

  const leaveCall = () => {
    clearTicket("patient");
    setStage("chat");
  };

  if (ticket) {
    return (
      <div className="mx-auto w-full max-w-4xl px-4 py-6">
        <VideoCall
          ticket={ticket}
          peerName="Doctor"
          waitingText="Waiting for a doctor to accept… Please stay on this page."
          onClose={leaveCall}
        />
        {ticket.consultation.summary && (
          <p className="mt-3 text-center text-sm text-muted-foreground">
            Your chat with the assistant was shared with the doctor.
          </p>
        )}
      </div>
    );
  }

  if (stage === "request") {
    return (
      <div className="mx-auto w-full max-w-md px-4 py-10">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-xl">
              <Stethoscope aria-hidden className="size-5 text-primary" />
              Talk to a volunteer doctor
            </CardTitle>
            <CardDescription>
              A doctor will join a video call with you. Keep this page open.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form
              className="flex flex-col gap-4"
              onSubmit={(event) => {
                event.preventDefault();
                requestDoctor();
              }}
            >
              <label className="flex flex-col gap-1.5 text-sm font-medium">
                Your name · तपाईंको नाम
                <Input
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  maxLength={60}
                  autoFocus
                  required
                  autoComplete="given-name"
                />
              </label>

              {messages.length > 0 && (
                <label className="flex items-start gap-2.5 text-sm">
                  <input
                    type="checkbox"
                    checked={share}
                    onChange={(event) => setShare(event.target.checked)}
                    className="mt-0.5 size-4 accent-[var(--primary)]"
                  />
                  <span>
                    Share my chat with the doctor, so I don&apos;t have to repeat myself.
                    <span className="block text-muted-foreground">
                      Only the doctor who accepts your call sees it.
                    </span>
                  </span>
                </label>
              )}

              {error && (
                <p role="alert" className="text-sm text-destructive">
                  {error}
                </p>
              )}

              <Button type="submit" size="lg" disabled={submitting || !name.trim()}>
                {submitting && <Loader2 aria-hidden className="animate-spin" />}
                Request a doctor
              </Button>
              <Button type="button" variant="ghost" onClick={() => setStage("chat")}>
                <ArrowLeft aria-hidden />
                Back to chat
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="mx-auto flex h-[calc(100dvh-3.5rem)] w-full max-w-3xl flex-col px-4 py-4">
      <ChatPanel onTalkToDoctor={() => setStage("request")} />
    </div>
  );
}
