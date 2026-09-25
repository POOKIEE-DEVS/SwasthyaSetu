"use client";

import { useEffect } from "react";
import { WifiOff } from "lucide-react";

import { useConnectionStore } from "@/lib/store/connection";

/**
 * Banner shown while the browser reports no network, so a dropped Wi-Fi
 * during the demo is obvious rather than looking like a broken app.
 */
export function ConnectionStatus() {
  const isOnline = useConnectionStore((state) => state.isOnline);
  const setOnline = useConnectionStore((state) => state.setOnline);

  useEffect(() => {
    setOnline(navigator.onLine);
    const goOnline = () => setOnline(true);
    const goOffline = () => setOnline(false);
    window.addEventListener("online", goOnline);
    window.addEventListener("offline", goOffline);
    return () => {
      window.removeEventListener("online", goOnline);
      window.removeEventListener("offline", goOffline);
    };
  }, [setOnline]);

  if (isOnline !== false) return null;

  return (
    <div
      role="status"
      aria-live="polite"
      className="sticky top-0 z-50 flex items-center justify-center gap-2 bg-triage-yellow px-4 py-2 text-sm font-medium text-triage-yellow-foreground"
    >
      <WifiOff aria-hidden className="size-4" />
      You&apos;re offline. Reconnect to chat or call. For an emergency, call 102.
    </div>
  );
}
