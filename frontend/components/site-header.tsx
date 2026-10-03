import Link from "next/link";
import { Phone } from "lucide-react";

import { AccountButton } from "@/components/account-button";
import { BrandLockup } from "@/components/brand";
import { GetHelpLink, HomeNavLinks, LanguageToggle } from "@/components/header-controls";
import { HeaderScrollState } from "@/components/motion/header-scroll-state";

type Props = {
  /** Small label for the current area on app pages ("Professional"). */
  role?: string;
  /** "home": the landing page's section links, language and main action. */
  variant?: "app" | "home";
  /** Show the English / नेपाली switch on an app page (the patient chat). */
  languageSwitch?: boolean;
};

/** The emergency number on app pages: always in the header, in red. */
function EmergencyButton() {
  return (
    <a
      href="tel:102"
      aria-label="Call 102 for an ambulance"
      className="flex h-11 items-center gap-1.5 rounded-[8px] bg-triage-red px-3 text-sm font-bold text-triage-red-foreground transition-colors duration-150 hover:bg-triage-red/90 sm:h-9"
    >
      <Phone aria-hidden className="size-4" />
      <span className="hidden sm:inline">Ambulance</span>
      <span className="tabular">102</span>
    </a>
  );
}

/** On the landing page 102 stays in view as you scroll, but quietly. */
function QuietEmergencyLink() {
  return (
    <a
      href="tel:102"
      aria-label="Call 102 for an ambulance"
      className="flex h-11 items-center gap-1.5 px-2 text-sm font-semibold text-foreground transition-colors duration-150 hover:text-triage-red-text"
    >
      <Phone aria-hidden className="size-4 text-triage-red-text" />
      <span className="tabular">102</span>
    </a>
  );
}

export function SiteHeader({ role, variant = "app", languageSwitch = false }: Props) {
  if (variant === "home") {
    return (
      <header className="site-header-home sticky top-0 z-40 border-b bg-background">
        <HeaderScrollState />
        <div className="mx-auto flex h-16 max-w-[1240px] items-center gap-10 px-4 sm:px-6">
          <Link href="/" aria-label="SwasthyaSetu home" className="rounded-[8px]">
            <BrandLockup />
          </Link>
          <nav aria-label="Sections" className="hidden lg:block">
            <HomeNavLinks />
          </nav>
          <div className="ml-auto flex items-center gap-1 sm:gap-5">
            <LanguageToggle />
            <QuietEmergencyLink />
            <GetHelpLink className="hidden md:inline-flex" />
          </div>
        </div>
      </header>
    );
  }

  return (
    <header className="sticky top-0 z-40 border-b bg-background">
      <div className="mx-auto flex h-16 max-w-[1240px] items-center justify-between gap-3 px-4 sm:px-6">
        <div className="flex min-w-0 items-center gap-3">
          <Link href="/" aria-label="SwasthyaSetu home" className="rounded-[8px]">
            <BrandLockup />
          </Link>
          {role && (
            <span className="hidden truncate rounded-[6px] border px-2 py-0.5 text-xs font-medium text-muted-foreground sm:inline">
              {role}
            </span>
          )}
        </div>

        <nav aria-label="Main" className="flex items-center gap-2 sm:gap-3">
          {languageSwitch && <LanguageToggle className="mr-2" />}
          <Link
            href="/doctor/"
            className="hidden rounded-[8px] px-2 py-1 text-sm font-medium text-muted-foreground transition-colors duration-150 hover:text-foreground md:inline"
          >
            For medical professionals
          </Link>
          <AccountButton />
          <EmergencyButton />
        </nav>
      </div>
    </header>
  );
}
