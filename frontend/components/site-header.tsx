import Link from "next/link";
import { Phone } from "lucide-react";

import { AccountButton } from "@/components/account-button";
import { BrandLockup } from "@/components/brand";
import {
  HomeNavLinks,
  LanguageToggle,
  TalkToProfessionalButton,
} from "@/components/header-controls";

type Props = {
  /** Small label for the current area on app pages ("Professional"). */
  role?: string;
  /** "home": the landing page's section links, language and main action. */
  variant?: "app" | "home";
};

/** The emergency number: always in the header, never waiting on anything. */
function EmergencyLink() {
  return (
    <a
      href="tel:102"
      aria-label="Call 102 for an ambulance"
      className="flex h-11 items-center gap-1.5 rounded-[12px] bg-triage-red px-3 text-sm font-bold text-triage-red-foreground transition-colors duration-150 hover:bg-triage-red/90 sm:h-9"
    >
      <Phone aria-hidden className="size-4" />
      <span className="hidden sm:inline">Ambulance</span>
      <span className="tabular">102</span>
    </a>
  );
}

export function SiteHeader({ role, variant = "app" }: Props) {
  return (
    <header className="sticky top-0 z-40 border-b bg-background/90 backdrop-blur-md">
      <div className="mx-auto flex h-16 max-w-[1240px] items-center justify-between gap-3 px-4 sm:px-6">
        <div className="flex min-w-0 items-center gap-3">
          <Link href="/" aria-label="SwasthyaSetu home" className="rounded-[8px]">
            <BrandLockup />
          </Link>
          {role && (
            <span className="hidden truncate rounded-[8px] border px-2 py-0.5 text-xs font-medium text-muted-foreground sm:inline">
              {role}
            </span>
          )}
        </div>

        {variant === "home" ? (
          <>
            <nav aria-label="Sections" className="hidden lg:block">
              <HomeNavLinks className="flex" />
            </nav>
            <div className="flex items-center gap-2">
              <LanguageToggle />
              <div className="hidden xl:block">
                <AccountButton />
              </div>
              <EmergencyLink />
              <TalkToProfessionalButton className="hidden md:inline-flex" />
            </div>
          </>
        ) : (
          <nav aria-label="Main" className="flex items-center gap-2 sm:gap-3">
            <Link
              href="/doctor/"
              className="hidden rounded-[8px] px-2 py-1 text-sm font-medium text-muted-foreground transition-colors duration-150 hover:text-foreground md:inline"
            >
              For medical professionals
            </Link>
            <AccountButton />
            <EmergencyLink />
          </nav>
        )}
      </div>
    </header>
  );
}
