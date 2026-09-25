import Link from "next/link";
import { Phone } from "lucide-react";

type Props = { role?: "Patient" | "Doctor" };

/** Brand, current role, and an always-visible emergency number. */
export function SiteHeader({ role }: Props) {
  return (
    <header className="sticky top-0 z-40 border-b bg-background/90 backdrop-blur">
      <div className="mx-auto flex h-14 max-w-6xl items-center justify-between gap-3 px-4">
        <Link href="/" className="flex items-baseline gap-2">
          <span className="font-semibold tracking-tight">SwasthyaSetu</span>
          <span className="hidden text-sm text-muted-foreground sm:inline">स्वास्थ्य सेतु</span>
          {role && (
            <span className="rounded-full bg-secondary px-2 py-0.5 text-xs font-medium text-secondary-foreground">
              {role}
            </span>
          )}
        </Link>
        {/* The emergency number is shown before anything else, and never
            waits on the AI or on a doctor being available. */}
        <a
          href="tel:102"
          className="flex items-center gap-1.5 rounded-md bg-triage-red px-3 py-1.5 text-sm font-semibold text-triage-red-foreground"
        >
          <Phone aria-hidden className="size-4" />
          Ambulance 102
        </a>
      </div>
    </header>
  );
}
