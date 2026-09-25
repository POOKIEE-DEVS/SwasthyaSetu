import { WifiOff } from "lucide-react";

import { Button } from "@/components/ui/button";

export const metadata = {
  title: "Offline",
};

/**
 * Served by the service worker when a navigation fails and the requested page
 * was never cached.
 *
 * It deliberately does not read like an error. Someone reaching this screen
 * may be mid-emergency, so it points at what still works rather than
 * apologising for what does not.
 */
export default function OfflinePage() {
  return (
    <main className="mx-auto flex min-h-dvh max-w-md flex-col justify-center gap-6 px-6 text-center">
      <WifiOff aria-hidden className="mx-auto size-10 text-triage-yellow" />
      <h1 className="text-2xl font-semibold tracking-tight">
        You are offline
      </h1>
      <p className="text-muted-foreground">
        First-aid guidance you have opened before, your emergency contacts, and
        your location are still available. Anything you record now is saved and
        sent as soon as you reconnect.
      </p>
      <div className="flex flex-col gap-3">
        <Button asChild size="lg" className="tap-target">
          <a href="/articles">Open saved first-aid guides</a>
        </Button>
        <Button asChild variant="outline" className="tap-target">
          <a href="/emergency-contacts">Emergency contacts</a>
        </Button>
      </div>
    </main>
  );
}
