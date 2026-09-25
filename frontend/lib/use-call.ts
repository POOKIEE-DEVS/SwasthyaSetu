"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import { api, wsUrl, type CallTicket } from "@/lib/api";

/**
 * One WebRTC video call between the patient and the doctor.
 *
 * Signalling goes over the backend WebSocket; audio and video go
 * peer-to-peer (or through TURN when no direct path exists).
 *
 * Offer rule: whoever is already in the room makes the offer when the
 * other participant arrives (they receive "peer-joined"). Only one side ever
 * gets that event, so both sides can never offer at once. The same rule
 * rebuilds the call after either side refreshes the page.
 */

export type CallStatus =
  | "starting" // getting camera/mic
  | "waiting" // in the room, other person not here yet
  | "connecting" // negotiating the peer connection
  | "connected"
  | "reconnecting" // media path dropped; WebRTC is retrying
  | "ended"
  | "failed";

type Signal =
  | { type: "joined"; role: string; peer_present: boolean }
  | { type: "peer-joined"; role: string }
  | { type: "peer-left"; role: string }
  | { type: "offer" | "answer"; payload: RTCSessionDescriptionInit }
  | { type: "ice-candidate"; payload: RTCIceCandidateInit }
  | { type: "hangup" }
  | { type: "error"; detail: string };

const CLOSE_REPLACED = 4000;
const CLOSE_UNAUTHORIZED = 4403;

// If a call hasn't connected this long after negotiation started, rejoin
// automatically. That covers a lost signalling message or a transient ICE
// failure. After MAX_AUTO_RETRIES the user gets a manual Reconnect button.
const CONNECT_TIMEOUT_MS = 20_000;
const MAX_AUTO_RETRIES = 2;

const VIDEO_CONSTRAINTS: MediaTrackConstraints = {
  width: { ideal: 640 },
  height: { ideal: 480 },
  facingMode: "user",
};

async function getMedia(): Promise<{ stream: MediaStream; hasVideo: boolean }> {
  if (!navigator.mediaDevices?.getUserMedia) {
    throw new Error(
      "Camera access needs a secure (https) connection. Open the app over https.",
    );
  }
  try {
    const stream = await navigator.mediaDevices.getUserMedia({
      audio: { echoCancellation: true, noiseSuppression: true },
      video: VIDEO_CONSTRAINTS,
    });
    return { stream, hasVideo: true };
  } catch (videoError) {
    // No camera, or it's busy (common with two browser windows on one
    // laptop): fall back to a voice-only call rather than failing.
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      return { stream, hasVideo: false };
    } catch {
      const name = (videoError as DOMException)?.name;
      throw new Error(
        name === "NotAllowedError"
          ? "Camera and microphone permission was denied. Allow it in the browser's address bar, then try again."
          : "Couldn't access a camera or microphone on this device.",
      );
    }
  }
}

export function useCall(ticket: CallTicket | null) {
  const [status, setStatus] = useState<CallStatus>("starting");
  const [message, setMessage] = useState<string | null>(null);
  const [localStream, setLocalStream] = useState<MediaStream | null>(null);
  const [remoteStream, setRemoteStream] = useState<MediaStream | null>(null);
  const [hasVideo, setHasVideo] = useState(true);
  const [micOn, setMicOn] = useState(true);
  const [cameraOn, setCameraOn] = useState(true);
  // Bumping this re-runs the whole join sequence (see `reconnect`).
  const [attempt, setAttempt] = useState(0);

  const wsRef = useRef<WebSocket | null>(null);
  const pcRef = useRef<RTCPeerConnection | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const pendingCandidates = useRef<RTCIceCandidateInit[]>([]);
  const endedRef = useRef(false);
  const autoRetries = useRef(0);

  /** Rejoin automatically, or give up and offer the manual button. */
  const retryOrFail = useCallback(() => {
    if (endedRef.current) return;
    if (autoRetries.current < MAX_AUTO_RETRIES) {
      autoRetries.current += 1;
      setMessage("Still connecting… retrying.");
      setAttempt((n) => n + 1);
    } else {
      setStatus("failed");
      setMessage(
        "Couldn't connect the call. Press Reconnect. If it keeps failing, the network may be blocking video calls.",
      );
    }
  }, []);

  // Watchdog: negotiation that stalls (a lost signal, ICE stuck checking)
  // never reaches "connected" on its own.
  useEffect(() => {
    if (status !== "connecting" && status !== "reconnecting") return;
    const timer = setTimeout(retryOrFail, CONNECT_TIMEOUT_MS);
    return () => clearTimeout(timer);
  }, [status, attempt, retryOrFail]);

  const send = useCallback((signal: object) => {
    const ws = wsRef.current;
    if (ws?.readyState === WebSocket.OPEN) ws.send(JSON.stringify(signal));
  }, []);

  const closePeer = useCallback(() => {
    pcRef.current?.close();
    pcRef.current = null;
    pendingCandidates.current = [];
    setRemoteStream(null);
  }, []);

  const teardown = useCallback(() => {
    closePeer();
    const ws = wsRef.current;
    wsRef.current = null;
    if (ws) {
      ws.onclose = null;
      ws.close();
    }
    streamRef.current?.getTracks().forEach((track) => track.stop());
    streamRef.current = null;
    setLocalStream(null);
  }, [closePeer]);

  const finish = useCallback(
    (reason: string, notifyPeer: boolean) => {
      if (endedRef.current) return;
      endedRef.current = true;
      if (notifyPeer) send({ type: "hangup" });
      if (ticket) api.end(ticket.consultation.id, ticket.token).catch(() => {});
      teardown();
      setMessage(reason);
      setStatus("ended");
    },
    [send, teardown, ticket],
  );

  useEffect(() => {
    if (!ticket) return;
    let cancelled = false;
    let reconnectAttempts = 0;
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
    endedRef.current = false;

    const iceServers: RTCIceServer[] = ticket.ice_servers.map((s) => ({
      urls: s.urls,
      ...(s.username ? { username: s.username } : {}),
      ...(s.credential ? { credential: s.credential } : {}),
    }));

    function newPeer(): RTCPeerConnection {
      closePeer();
      const pc = new RTCPeerConnection({ iceServers });
      const stream = streamRef.current;
      stream?.getTracks().forEach((track) => pc.addTrack(track, stream));

      pc.onicecandidate = (event) => {
        if (event.candidate) send({ type: "ice-candidate", payload: event.candidate.toJSON() });
      };
      pc.ontrack = (event) => {
        setRemoteStream(event.streams[0] ?? new MediaStream([event.track]));
      };
      pc.onconnectionstatechange = () => {
        if (pcRef.current !== pc) return;
        switch (pc.connectionState) {
          case "connected":
            autoRetries.current = 0;
            setStatus("connected");
            setMessage(null);
            break;
          case "disconnected":
            // Often recovers by itself; the watchdog rejoins if it doesn't.
            setStatus("reconnecting");
            setMessage("Connection unstable. Trying to recover…");
            break;
          case "failed":
            // No working network path right now. Rejoin at once rather than
            // waiting for the watchdog.
            retryOrFail();
            break;
        }
      };
      pcRef.current = pc;
      return pc;
    }

    async function flushCandidates(pc: RTCPeerConnection) {
      const queued = pendingCandidates.current;
      pendingCandidates.current = [];
      for (const candidate of queued) {
        await pc.addIceCandidate(candidate).catch(() => {});
      }
    }

    async function onSignal(signal: Signal) {
      switch (signal.type) {
        case "joined":
          setStatus(signal.peer_present ? "connecting" : "waiting");
          break;

        case "peer-joined": {
          // We were here first: make the offer.
          setStatus("connecting");
          setMessage(null);
          const pc = newPeer();
          await pc.setLocalDescription(await pc.createOffer());
          send({ type: "offer", payload: pc.localDescription!.toJSON() });
          break;
        }

        case "offer": {
          setStatus("connecting");
          const pc = newPeer();
          await pc.setRemoteDescription(signal.payload);
          await flushCandidates(pc);
          await pc.setLocalDescription(await pc.createAnswer());
          send({ type: "answer", payload: pc.localDescription!.toJSON() });
          break;
        }

        case "answer": {
          const pc = pcRef.current;
          if (!pc || pc.signalingState !== "have-local-offer") return;
          await pc.setRemoteDescription(signal.payload);
          await flushCandidates(pc);
          break;
        }

        case "ice-candidate": {
          const pc = pcRef.current;
          // Candidates can arrive before the description they belong to.
          if (!pc || !pc.remoteDescription) {
            pendingCandidates.current.push(signal.payload);
          } else {
            await pc.addIceCandidate(signal.payload).catch(() => {});
          }
          break;
        }

        case "peer-left":
          closePeer();
          setStatus("waiting");
          setMessage(
            ticket!.role === "patient"
              ? "The doctor lost connection. Waiting for them to rejoin…"
              : "The patient lost connection. Waiting for them to rejoin…",
          );
          break;

        case "hangup":
          finish(
            ticket!.role === "patient"
              ? "The doctor ended the call."
              : "The patient ended the call.",
            false,
          );
          break;
      }
    }

    function connect() {
      const ws = new WebSocket(
        wsUrl(`/ws/consultations/${ticket!.consultation.id}?token=${encodeURIComponent(ticket!.token)}`),
      );
      wsRef.current = ws;

      ws.onopen = () => {
        reconnectAttempts = 0;
      };
      ws.onmessage = (event) => {
        onSignal(JSON.parse(event.data) as Signal).catch((error) => {
          console.error("signalling error", error);
        });
      };
      ws.onclose = (event) => {
        if (cancelled || endedRef.current || wsRef.current !== ws) return;
        if (event.code === CLOSE_REPLACED) return; // this call is open in another tab
        if (event.code === CLOSE_UNAUTHORIZED) {
          endedRef.current = true;
          teardown();
          setStatus("ended");
          setMessage("This call has ended.");
          return;
        }
        if (reconnectAttempts >= 6) {
          setStatus("failed");
          setMessage("Lost connection to the server.");
          return;
        }
        reconnectAttempts += 1;
        setMessage("Reconnecting to the server…");
        reconnectTimer = setTimeout(connect, Math.min(1000 * 2 ** reconnectAttempts, 8000));
      };
    }

    // A camera track can end mid-call: USB webcam unplugged, another app
    // grabbing the camera, an OS privacy toggle, two windows on one laptop.
    // Re-acquire it and hot-swap it into the live call (replaceTrack needs no
    // renegotiation). If that fails, carry on as voice-only.
    function watchCamera(track: MediaStreamTrack) {
      track.onended = async () => {
        const stream = streamRef.current;
        if (cancelled || endedRef.current || !stream) return;
        try {
          const fresh = await navigator.mediaDevices.getUserMedia({ video: VIDEO_CONSTRAINTS });
          const replacement = fresh.getVideoTracks()[0];
          if (cancelled || endedRef.current || streamRef.current !== stream) {
            replacement.stop();
            return;
          }
          replacement.enabled = track.enabled; // keep the user's camera on/off choice
          stream.removeTrack(track);
          stream.addTrack(replacement);
          const sender = pcRef.current
            ?.getSenders()
            .find((s) => s.track === track || s.track?.kind === "video");
          await sender?.replaceTrack(replacement);
          // A new stream object makes the self-view pick up the new track.
          setLocalStream(new MediaStream(stream.getTracks()));
          watchCamera(replacement);
        } catch {
          if (cancelled) return;
          setHasVideo(false);
          setCameraOn(false);
        }
      };
    }

    (async () => {
      setStatus("starting");
      setMessage(null);
      try {
        const media = await getMedia();
        if (cancelled) {
          media.stream.getTracks().forEach((track) => track.stop());
          return;
        }
        streamRef.current = media.stream;
        setLocalStream(media.stream);
        setHasVideo(media.hasVideo);
        setCameraOn(media.hasVideo);
        setMicOn(true);
        const camera = media.stream.getVideoTracks()[0];
        if (camera) watchCamera(camera);
        connect();
      } catch (error) {
        if (cancelled) return;
        setStatus("failed");
        setMessage((error as Error).message);
      }
    })();

    return () => {
      // Unmount only (including React's dev double-mount). This must not end
      // the consultation: the server replaces this socket when we reconnect.
      cancelled = true;
      clearTimeout(reconnectTimer);
      teardown();
    };
  }, [ticket, attempt, send, closePeer, teardown, finish, retryOrFail]);

  /** Rejoin from scratch. The other side sees "peer-joined" and re-offers. */
  const reconnect = useCallback(() => {
    autoRetries.current = 0;
    setAttempt((n) => n + 1);
  }, []);

  const toggleMic = useCallback(() => {
    const track = streamRef.current?.getAudioTracks()[0];
    if (!track) return;
    track.enabled = !track.enabled;
    setMicOn(track.enabled);
  }, []);

  const toggleCamera = useCallback(() => {
    const track = streamRef.current?.getVideoTracks()[0];
    if (!track) return;
    track.enabled = !track.enabled;
    setCameraOn(track.enabled);
  }, []);

  const hangUp = useCallback(() => finish("You ended the call.", true), [finish]);

  return {
    status,
    message,
    localStream,
    remoteStream,
    hasVideo,
    micOn,
    cameraOn,
    toggleMic,
    toggleCamera,
    hangUp,
    reconnect,
  };
}
