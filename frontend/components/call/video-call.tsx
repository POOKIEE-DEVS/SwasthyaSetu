"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Loader2, Mic, MicOff, PhoneOff, RefreshCw, Video, VideoOff, Volume2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import type { CallTicket } from "@/lib/api";
import { useCall, type CallStatus } from "@/lib/use-call";
import { cn } from "@/lib/utils";

const STATUS_TEXT: Record<CallStatus, string> = {
  starting: "Starting camera…",
  waiting: "Waiting…",
  connecting: "Connecting…",
  connected: "Connected",
  reconnecting: "Reconnecting…",
  ended: "Call ended",
  failed: "Connection problem",
};

function useVideo(stream: MediaStream | null, onAutoplayBlocked?: () => void) {
  const ref = useRef<HTMLVideoElement>(null);
  useEffect(() => {
    const video = ref.current;
    if (!video) return;
    video.srcObject = stream;
    if (!stream) return;
    video.play().catch(() => {
      // Browsers can block autoplay with sound. Play muted so the video is
      // still visible, and let the user tap to turn the sound on.
      video.muted = true;
      video.play().catch(() => {});
      onAutoplayBlocked?.();
    });
  }, [stream, onAutoplayBlocked]);
  return ref;
}

type Props = {
  ticket: CallTicket;
  peerName: string;
  /** Shown while no one else is in the call yet. */
  waitingText: string;
  onClose: () => void;
};

export function VideoCall({ ticket, peerName, waitingText, onClose }: Props) {
  const call = useCall(ticket);
  const [soundBlocked, setSoundBlocked] = useState(false);
  // Stable identity, so the video effect doesn't re-run on every render.
  const onSoundBlocked = useCallback(() => setSoundBlocked(true), []);
  const localRef = useVideo(call.localStream);
  const remoteRef = useVideo(call.remoteStream, onSoundBlocked);

  if (call.status === "ended") {
    return (
      <div className="flex flex-col items-center gap-4 rounded-xl border bg-card p-10 text-center">
        <PhoneOff aria-hidden className="size-8 text-muted-foreground" />
        <p className="text-lg font-medium">{call.message ?? "Call ended"}</p>
        <Button onClick={onClose}>Done</Button>
      </div>
    );
  }

  const peerVisible = call.status === "connected" && call.remoteStream;

  return (
    <div className="flex flex-col gap-3">
      <div className="relative aspect-video w-full overflow-hidden rounded-xl bg-neutral-900">
        <video
          ref={remoteRef}
          autoPlay
          playsInline
          className={cn("size-full object-cover", !peerVisible && "invisible")}
        />

        {!peerVisible && (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 p-6 text-center text-white">
            {call.status !== "failed" && <Loader2 aria-hidden className="size-8 animate-spin" />}
            <p className="text-lg font-medium">
              {call.status === "waiting" && !call.message ? waitingText : STATUS_TEXT[call.status]}
            </p>
            {call.message && <p className="max-w-md text-sm text-white/80">{call.message}</p>}
            {call.status === "failed" && (
              <Button variant="secondary" onClick={call.reconnect}>
                <RefreshCw aria-hidden />
                Reconnect
              </Button>
            )}
          </div>
        )}

        {peerVisible && (
          <span className="absolute left-3 top-3 rounded-md bg-black/60 px-2 py-1 text-sm text-white">
            {peerName}
          </span>
        )}

        {soundBlocked && peerVisible && (
          <button
            type="button"
            onClick={() => {
              const video = remoteRef.current;
              if (!video) return;
              video.muted = false;
              video.play().then(() => setSoundBlocked(false)).catch(() => {});
            }}
            className="absolute inset-x-0 bottom-4 mx-auto flex w-fit items-center gap-2 rounded-full bg-white px-4 py-2 text-sm font-medium text-black"
          >
            <Volume2 aria-hidden className="size-4" />
            Tap to turn on sound
          </button>
        )}

        {/* Own camera, picture-in-picture. Muted so it never echoes. */}
        <div className="absolute bottom-3 right-3 aspect-video w-28 overflow-hidden rounded-lg border border-white/30 bg-neutral-800 sm:w-44">
          <video
            ref={localRef}
            autoPlay
            playsInline
            muted
            className={cn(
              "size-full -scale-x-100 object-cover",
              (!call.hasVideo || !call.cameraOn) && "invisible",
            )}
          />
          {(!call.hasVideo || !call.cameraOn) && (
            <div className="absolute inset-0 flex items-center justify-center text-xs text-white/70">
              {call.hasVideo ? "Camera off" : "Voice only"}
            </div>
          )}
        </div>
      </div>

      <div className="flex items-center justify-center gap-3">
        <Button
          variant={call.micOn ? "secondary" : "destructive"}
          size="lg"
          onClick={call.toggleMic}
          aria-pressed={!call.micOn}
          aria-label={call.micOn ? "Mute microphone" : "Unmute microphone"}
        >
          {call.micOn ? <Mic aria-hidden /> : <MicOff aria-hidden />}
          {call.micOn ? "Mute" : "Unmute"}
        </Button>
        <Button
          variant={call.cameraOn ? "secondary" : "destructive"}
          size="lg"
          onClick={call.toggleCamera}
          disabled={!call.hasVideo}
          aria-pressed={!call.cameraOn}
          aria-label={call.cameraOn ? "Turn camera off" : "Turn camera on"}
        >
          {call.cameraOn ? <Video aria-hidden /> : <VideoOff aria-hidden />}
          {call.cameraOn ? "Camera" : "Camera off"}
        </Button>
        <Button variant="emergency" size="lg" onClick={call.hangUp}>
          <PhoneOff aria-hidden />
          {call.status === "waiting" && ticket.role === "patient" ? "Cancel" : "End call"}
        </Button>
      </div>
    </div>
  );
}
