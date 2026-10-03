import Link from "next/link";
import { Phone } from "lucide-react";

import { AccountButton } from "@/components/account-button";
import { BrandLockup } from "@/components/brand";

type Props = { role?: string };

/** Brand, current area, account, and an always-visible emergency number. */
export function SiteHeader({ role }: Props) {
  return (
    <header className="sticky top-0 z-40 border-b border-border/70 bg-background/80 backdrop-blur-md">
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between gap-3 px-4">
        <div className="flex min-w-0 items-center gap-3">
          <Link href="/" aria-label="SwasthyaSetu home" className="rounded-full">
            <BrandLockup />
          </Link>
          {role && (
            <span className="hidden truncate rounded-full border px-2.5 py-0.5 text-xs font-medium text-muted-foreground sm:inline">
              {role}
            </span>
          )}
        </div>
        <nav aria-label="Main" className="flex items-center gap-2 sm:gap-3">
          <Link
            href="/doctor/"
            className="hidden rounded-full px-2 py-1 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground md:inline"
          >
            For medical professionals
          </Link>
          <AccountButton />
          {/* The emergency number is shown before anything else, and never
              waits on the AI, a login, or a doctor being available. */}
          <a
            href="tel:102"
            aria-label="Call 102 for an ambulance"
            className="flex h-9 items-center gap-1.5 rounded-full bg-triage-red px-3.5 text-sm font-bold text-triage-red-foreground shadow-soft transition-colors hover:bg-triage-red/90"
          >
            <Phone aria-hidden className="size-4" />
            <span className="hidden sm:inline">Ambulance</span>
            <span className="tabular">102</span>
          </a>
        </nav>
      </div>
    </header>
  );
}
