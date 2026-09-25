"use client";

import { useEffect } from "react";
import { WifiOff } from "lucide-react";

import { useConnectionStore } from "@/lib/store/connection";
import { cn } from "@/lib/utils";

/**
 * Watches the browser's connectivity and mirrors it into the Zustand store.
 *
 * Rendered once in the root layout. It shows nothing while online, and a
 * persistent banner when offline -- silence would leave a user wondering
 * whether the app is broken at exactly the moment they need to trust it.
 */
export function ConnectionStatus() {
  const isOnline = useConnectionStore((state) => state.isOnline);
  const queuedReports = useConnectionStore((state) => state.queuedReports);
  const setOnline = useConnectionStore((state) => state.setOnline);

  useEffect(() => {
    // Read once on mount rather than during render: `navigator` does not
    // exist during server rendering, and reading it in render would produce
    // a hydration mismatch.
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

  // `null` means "not yet determined"; assume online so the banner never
  // flashes on a good connection.
  if (isOnline !== false) return null;

  return (
    <div
      role="status"
      aria-live="polite"
      className={cn(
        "sticky top-0 z-50 flex items-center justify-center gap-2",
        "bg-triage-yellow px-4 py-2 text-sm font-medium",
        "text-triage-yellow-foreground",
      )}
    >
      <WifiOff aria-hidden className="size-4" />
      <span>
        Offline — saved first-aid guidance and emergency contacts are still
        available.
        {queuedReports > 0 &&
          ` ${queuedReports} report${queuedReports === 1 ? "" : "s"} will sync when you reconnect.`}
      </span>
    </div>
  );
}
