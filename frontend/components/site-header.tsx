import Link from "next/link";
import { Phone } from "lucide-react";

import { AccountButton } from "@/components/account-button";

type Props = { role?: string };

/** Brand, current area, account, and an always-visible emergency number. */
export function SiteHeader({ role }: Props) {
  return (
    <header className="sticky top-0 z-40 border-b bg-background/90 backdrop-blur">
      <div className="mx-auto flex h-14 max-w-6xl items-center justify-between gap-3 px-4">
        <Link href="/" className="flex min-w-0 items-baseline gap-2">
          <span className="font-semibold tracking-tight">SwasthyaSetu</span>
          <span className="hidden text-sm text-muted-foreground sm:inline">स्वास्थ्य सेतु</span>
          {role && (
            <span className="truncate rounded-full bg-secondary px-2 py-0.5 text-xs font-medium text-secondary-foreground">
              {role}
            </span>
          )}
        </Link>
        <div className="flex items-center gap-2">
          <Link
            href="/doctor/"
            className="hidden text-sm text-muted-foreground hover:text-foreground md:inline"
          >
            For medical professionals
          </Link>
          <AccountButton />
          {/* The emergency number is shown before anything else, and never
              waits on the AI, a login, or a doctor being available. */}
          <a
            href="tel:102"
            className="flex items-center gap-1.5 rounded-md bg-triage-red px-3 py-1.5 text-sm font-semibold text-triage-red-foreground"
          >
            <Phone aria-hidden className="size-4" />
            <span className="hidden sm:inline">Ambulance</span> 102
          </a>
        </div>
      </div>
    </header>
  );
}
