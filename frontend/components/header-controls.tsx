"use client";

import Link from "next/link";

import { setLang, useCopy } from "@/lib/i18n";
import { cn } from "@/lib/utils";

/**
 * English / नेपाली, as two words rather than a pill: the current one in
 * ink and underlined. Changes the page and chat text, not the replies.
 */
export function LanguageToggle({ className }: { className?: string }) {
  const { lang, t } = useCopy();
  const option = (value: "en" | "ne", label: string) => (
    <button
      type="button"
      lang={value}
      aria-pressed={lang === value}
      onClick={() => setLang(value)}
      className={cn(
        "min-h-9 text-sm font-medium underline-offset-[6px] transition-colors duration-150",
        lang === value
          ? "text-foreground underline decoration-primary decoration-[1.5px]"
          : "text-muted-foreground hover:text-foreground",
      )}
    >
      {label}
    </button>
  );
  return (
    <>
      <div
        role="group"
        aria-label={t.nav.language}
        className={cn("hidden items-center gap-4 sm:flex", className)}
      >
        {option("en", "English")}
        {option("ne", "नेपाली")}
      </div>
      {/* Phones: one word that switches to the other language, so the
          header still fits beside the logo and 102. */}
      <button
        type="button"
        lang={lang === "en" ? "ne" : "en"}
        onClick={() => setLang(lang === "en" ? "ne" : "en")}
        aria-label={lang === "en" ? "नेपालीमा हेर्नुहोस् (Switch to Nepali)" : "Switch to English"}
        className="flex h-11 items-center px-2 text-sm font-medium text-foreground sm:hidden"
      >
        {lang === "en" ? "नेपाली" : "English"}
      </button>
    </>
  );
}

/** "Get help": the first-aid chat, no sign-in. */
export function GetHelpLink({ className }: { className?: string }) {
  const { t } = useCopy();
  return (
    <Link
      href="/patient/"
      className={cn(
        "inline-flex h-10 items-center rounded-[8px] bg-primary px-4 text-sm font-semibold text-primary-foreground transition-colors duration-150 hover:bg-primary/90 active:translate-y-px",
        className,
      )}
    >
      {t.nav.getHelp}
    </Link>
  );
}

/** Section links for the landing page: plain words, no containers. */
export function HomeNavLinks({ className }: { className?: string }) {
  const { t } = useCopy();
  const links: [string, string][] = [
    ["#how", t.nav.how],
    ["#professionals", t.nav.professionals],
    ["#safety", t.nav.safety],
  ];
  return (
    <ul className={cn("flex items-center gap-7", className)}>
      {links.map(([href, label]) => (
        <li key={href}>
          <a
            href={href}
            className="text-sm text-muted-foreground transition-colors duration-150 hover:text-foreground"
          >
            {label}
          </a>
        </li>
      ))}
    </ul>
  );
}
