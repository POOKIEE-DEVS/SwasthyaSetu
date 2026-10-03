"use client";

import { setLang, useCopy } from "@/lib/i18n";
import { openTalkToProfessional } from "@/lib/store/patient-ui";
import { cn } from "@/lib/utils";

/** English / नेपाली switch. Changes the page's own text, not the chat. */
export function LanguageToggle({ className }: { className?: string }) {
  const { lang, t } = useCopy();
  const option = (value: "en" | "ne", label: string, langAttr: string) => (
    <button
      type="button"
      lang={langAttr}
      aria-pressed={lang === value}
      onClick={() => setLang(value)}
      className={cn(
        "min-h-11 rounded-[8px] px-2.5 text-[13px] font-semibold transition-colors duration-150 sm:min-h-8",
        lang === value
          ? "bg-card text-foreground shadow-soft"
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
        className={cn(
          "hidden items-center gap-0.5 rounded-[10px] bg-muted p-0.5 sm:flex",
          className,
        )}
      >
        {option("en", "English", "en")}
        {option("ne", "नेपाली", "ne")}
      </div>
      {/* Phones: one button that switches to the other language, so the
          header still fits beside the logo and the 102 button. */}
      <button
        type="button"
        lang={lang === "en" ? "ne" : "en"}
        onClick={() => setLang(lang === "en" ? "ne" : "en")}
        aria-label={lang === "en" ? "नेपालीमा हेर्नुहोस् (Switch to Nepali)" : "Switch to English"}
        className="flex h-11 items-center rounded-[10px] bg-muted px-3 text-[13px] font-semibold text-foreground sm:hidden"
      >
        {lang === "en" ? "नेपाली" : "English"}
      </button>
    </>
  );
}

/** "Talk to a professional": opens the request form on the home page. */
export function TalkToProfessionalButton({ className }: { className?: string }) {
  const { t } = useCopy();
  return (
    <button
      type="button"
      onClick={openTalkToProfessional}
      className={cn(
        "inline-flex h-10 items-center rounded-[12px] bg-primary px-4 text-sm font-semibold text-primary-foreground transition-colors duration-150 hover:bg-primary/90 active:translate-y-px",
        className,
      )}
    >
      {t.nav.talk}
    </button>
  );
}

/** Section links for the landing page. */
export function HomeNavLinks({ className }: { className?: string }) {
  const { t } = useCopy();
  const links: [string, string][] = [
    ["#how", t.nav.how],
    ["#professionals", t.nav.professionals],
    ["#safety", t.nav.safety],
  ];
  return (
    <ul className={cn("items-center gap-1", className)}>
      {links.map(([href, label]) => (
        <li key={href}>
          <a
            href={href}
            className="rounded-[8px] px-3 py-2 text-sm font-medium text-muted-foreground transition-colors duration-150 hover:text-foreground"
          >
            {label}
          </a>
        </li>
      ))}
    </ul>
  );
}
