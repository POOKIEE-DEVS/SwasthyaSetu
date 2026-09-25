"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Loader2, Power, UserRound } from "lucide-react";

import { VideoCall } from "@/components/call/video-call";
import { MessageText } from "@/components/chat/message-text";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { api, ApiError, wsUrl, type Consultation } from "@/lib/api";
import { clearTicket, saveTicket, useTicket } from "@/lib/session";

/** Short alert tone for a new waiting patient. Needs a prior user click. */
function beep(ctx: AudioContext | null) {
  if (!ctx) return;
  const osc = ctx.createOscillator();
  const gain = ctx.createGain();
  osc.frequency.value = 880;
  gain.gain.setValueAtTime(0.15, ctx.currentTime);
  gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.4);
  osc.connect(gain).connect(ctx.destination);
  osc.start();
  osc.stop(ctx.currentTime + 0.4);
}

function waitedFor(createdAt: number, now: number): string {
  const minutes = Math.floor((now / 1000 - createdAt) / 60);
  return minutes < 1 ? "just now" : `${minutes} min`;
}

/** Live queue over /ws/doctors, reconnecting if the connection drops. */
function useQueue(enabled: boolean, onNewPatient: () => void) {
  const [queue, setQueue] = useState<Consultation[]>([]);
  const [connected, setConnected] = useState(false);
  const knownIds = useRef<Set<string> | null>(null);

  useEffect(() => {
    if (!enabled) return;
    let ws: WebSocket | null = null;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let attempts = 0;
    let stopped = false;

    const connect = () => {
      ws = new WebSocket(wsUrl("/ws/doctors"));
      ws.onopen = () => {
        attempts = 0;
        setConnected(true);
      };
      ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        if (data.type !== "queue") return;
        const items = data.consultations as Consultation[];
        const ids = new Set(items.map((c) => c.id));
        // Alert on arrivals, but not for the initial snapshot.
        if (knownIds.current && items.some((c) => !knownIds.current!.has(c.id))) {
          onNewPatient();
        }
        knownIds.current = ids;
        setQueue(items);
      };
      ws.onclose = () => {
        setConnected(false);
        if (stopped) return;
        attempts += 1;
        timer = setTimeout(connect, Math.min(1000 * 2 ** attempts, 10000));
      };
    };
    connect();

    return () => {
      stopped = true;
      clearTimeout(timer);
      ws?.close();
      knownIds.current = null;
    };
  }, [enabled, onNewPatient]);

  return { queue, connected };
}

export function DoctorView() {
  const [wentOnline, setOnline] = useState(false);
  const ticket = useTicket("doctor");
  // A call restored after a refresh implies the doctor was online.
  const online = wentOnline || ticket !== null;
  const [acceptingId, setAcceptingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [now, setNow] = useState(() => Date.now());
  const audioRef = useRef<AudioContext | null>(null);

  const onNewPatient = useCallback(() => beep(audioRef.current), []);
  const { queue, connected } = useQueue(online && !ticket, onNewPatient);

  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 15000);
    return () => clearInterval(id);
  }, []);

  useEffect(() => {
    document.title = queue.length > 0 ? `(${queue.length}) Waiting · SwasthyaSetu` : "Doctor · SwasthyaSetu";
  }, [queue.length]);

  const goOnline = () => {
    // Created inside a click, so the browser allows it to play sound later.
    audioRef.current ??= new AudioContext();
    setOnline(true);
  };

  const accept = async (id: string) => {
    setAcceptingId(id);
    setError(null);
    try {
      const newTicket = await api.accept(id);
      saveTicket(newTicket);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Couldn't accept this request.");
    } finally {
      setAcceptingId(null);
    }
  };

  const leaveCall = () => {
    clearTicket("doctor");
  };

  if (ticket) {
    const { patient_name, summary } = ticket.consultation;
    return (
      <div className="mx-auto grid w-full max-w-6xl gap-4 px-4 py-6 lg:grid-cols-[1fr_22rem]">
        <VideoCall
          ticket={ticket}
          peerName={patient_name}
          waitingText={`Connecting to ${patient_name}…`}
          onClose={leaveCall}
        />
        <Card className="h-fit lg:max-h-[calc(100dvh-7rem)] lg:overflow-y-auto">
          <CardHeader>
            <CardTitle>{patient_name}</CardTitle>
            <CardDescription>
              {summary ? "Shared their chat with the AI assistant:" : "Did not share a chat."}
            </CardDescription>
          </CardHeader>
          {summary && (
            <CardContent className="text-sm">
              <MessageText text={summary} />
            </CardContent>
          )}
        </Card>
      </div>
    );
  }

  if (!online) {
    return (
      <div className="mx-auto w-full max-w-md px-4 py-16">
        <Card>
          <CardHeader>
            <CardTitle className="text-xl">Volunteer doctor</CardTitle>
            <CardDescription>
              Go online to see patients who are waiting and take their video calls.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button size="lg" className="w-full" onClick={goOnline}>
              <Power aria-hidden />
              Go online
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="mx-auto w-full max-w-3xl px-4 py-6">
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold">Waiting patients</h1>
          <p className="flex items-center gap-2 text-sm text-muted-foreground">
            <span
              className={`inline-block size-2 rounded-full ${connected ? "bg-triage-green" : "bg-triage-yellow"}`}
            />
            {connected ? "Online · you'll hear a tone when a patient arrives" : "Connecting…"}
          </p>
        </div>
        <Button variant="outline" onClick={() => setOnline(false)}>
          Go offline
        </Button>
      </div>

      {error && (
        <p role="alert" className="mb-3 rounded-lg border border-destructive/40 bg-destructive/5 px-3 py-2 text-sm">
          {error}
        </p>
      )}

      {queue.length === 0 ? (
        <div className="rounded-xl border border-dashed p-10 text-center text-muted-foreground">
          No patients waiting right now.
        </div>
      ) : (
        <ul className="flex flex-col gap-3">
          {queue.map((c) => (
            <li key={c.id}>
              <Card>
                <CardHeader className="flex-row items-center justify-between gap-4 space-y-0">
                  <div className="flex items-center gap-3">
                    <UserRound aria-hidden className="size-8 rounded-full bg-secondary p-1.5" />
                    <div>
                      <CardTitle>{c.patient_name}</CardTitle>
                      <CardDescription>Waiting {waitedFor(c.created_at, now)}</CardDescription>
                    </div>
                  </div>
                  <Button onClick={() => accept(c.id)} disabled={acceptingId !== null}>
                    {acceptingId === c.id && <Loader2 aria-hidden className="animate-spin" />}
                    Accept
                  </Button>
                </CardHeader>
                {c.summary && (
                  <CardContent>
                    <details className="text-sm">
                      <summary className="cursor-pointer text-muted-foreground">
                        Read their chat with the assistant
                      </summary>
                      <div className="mt-2 max-h-60 overflow-y-auto rounded-md bg-muted p-3">
                        <MessageText text={c.summary} />
                      </div>
                    </details>
                  </CardContent>
                )}
              </Card>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
