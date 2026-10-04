"use client";

import type { CSSProperties } from "react";
import Link from "next/link";
import { ArrowRight, Phone } from "lucide-react";

import { BrandLockup } from "@/components/brand";
import { ExampleExchange } from "@/components/chat/example-exchange";
import { EditorialPhoto } from "@/components/home/editorial-photo";
import { SetuBridge } from "@/components/home/setu-bridge";
import { HeroIntro } from "@/components/motion/hero-intro";
import { RevealObserver } from "@/components/motion/reveal-observer";
import { StringTuneInit } from "@/components/motion/string-tune-init";
import { Button } from "@/components/ui/button";
import { useCopy } from "@/lib/i18n";
import { cn } from "@/lib/utils";

const CONTAINER = "mx-auto w-full max-w-[1240px] px-4 sm:px-6";
/** Where each journey stage comes forward along the scroll (0 to 1). */
const STAGE_AT = [0.08, 0.36, 0.7];
/** Sections breathe: 96px on phones, up to 160px on large screens. */
const SECTION_Y = "py-24 lg:py-32 xl:py-40";

/** Custom properties for the motion CSS (--step, --after, …). */
function vars(values: Record<string, string | number>): CSSProperties {
  return values as CSSProperties;
}

/** The arrow that leans forward on hover; the text stays put. */
function NudgeArrow() {
  return (
    <ArrowRight
      aria-hidden
      className="size-4 transition-[translate] duration-150 ease-out group-hover:translate-x-[3px]"
    />
  );
}

/** Several short lines set as one headline, each starting a new line. */
function Lines({ lines }: { lines: string[] }) {
  return lines.map((line) => (
    <span key={line} className="block">
      {line}
    </span>
  ));
}

/** The page's one product action: the first-aid chat, no sign-in. */
function GetHelpButton({ label }: { label: string }) {
  return (
    <Button asChild size="lg" className="group h-12 px-6">
      <Link href="/patient/">
        {label}
        <NudgeArrow />
      </Link>
    </Button>
  );
}

/**
 * 102, always one tap away but never shouting: ink text, a red phone.
 * On phones it becomes a full-width target right under the main action.
 */
function EmergencyCall({ className }: { className?: string }) {
  const { t } = useCopy();
  return (
    <a
      href="tel:102"
      className={cn(
        "inline-flex min-h-12 items-center justify-center gap-2.5 whitespace-nowrap rounded-[8px] border border-input px-5 text-[15px] text-foreground transition-colors duration-150 hover:border-foreground/40 sm:justify-start sm:border-0 sm:px-0",
        className,
      )}
    >
      <Phone aria-hidden className="size-4 text-triage-red-text" />
      <span>
        {t.hero.emergencyQ}{" "}
        <span className="font-semibold underline decoration-triage-red-text/50 underline-offset-4">
          {t.hero.emergencyCall}
        </span>
      </span>
    </a>
  );
}

function Hero() {
  const { t } = useCopy();
  return (
    <section
      aria-labelledby="hero-title"
      className={`hero ${CONTAINER} pb-20 pt-14 sm:pt-20 lg:pb-24 lg:pt-28`}
    >
      <h1 id="hero-title" className="font-display text-display">
        {/* Screen readers get the sentence once, as written. The visual
            lines are split by StringSplit on desktop and rise in. */}
        <span className="sr-only">{t.hero.title.join(" ")}</span>
        {t.hero.title.map((line, i) => (
          <span
            key={line}
            aria-hidden
            className="hero-line block"
            data-string="split"
            data-string-split="line"
            style={vars({ "--line": i })}
          >
            {line}
          </span>
        ))}
      </h1>

      <div className="mt-8 lg:mt-14 lg:grid lg:grid-cols-12 lg:gap-6">
        <div className="lg:col-span-6 lg:col-start-7 xl:col-span-5 xl:col-start-7">
          <p
            className="hero-after max-w-[36ch] text-lede text-muted-foreground"
            style={vars({ "--after": "250ms" })}
          >
            {t.hero.lede}
          </p>
          <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center sm:gap-x-8">
            <div className="hero-after flex flex-col sm:block" style={vars({ "--after": "400ms" })}>
              <GetHelpButton label={t.hero.cta} />
            </div>
            {/* 102 is never part of the entrance: it is there from the
                first paint. */}
            <EmergencyCall />
          </div>
        </div>
      </div>

      <SetuBridge className="mt-20 lg:mt-28" />
    </section>
  );
}

/** Why SwasthyaSetu exists, before what it does. */
function Why() {
  const { t } = useCopy();
  return (
    <section aria-labelledby="why-title" className="bg-paper">
      <div className={`${CONTAINER} ${SECTION_Y}`}>
        {/* The slowest reveal on the page: this is the emotional beat. */}
        <div
          data-reveal="group"
          className="grid gap-6 lg:grid-cols-12"
          style={vars({ "--reveal-duration": "950ms", "--stagger": "160ms" })}
        >
          <p
            data-reveal-step
            className="max-w-[24ch] font-medium text-muted-foreground lg:col-span-3 lg:pt-5"
          >
            {t.why.lead}
          </p>
          <div className="lg:col-span-8 lg:col-start-5">
            <h2
              id="why-title"
              data-reveal-step
              className="font-display text-headline"
              style={vars({ "--step": 1 })}
            >
              {t.why.title}
            </h2>
            <p
              data-reveal-step
              className="mt-8 max-w-[44ch] text-lede text-foreground/80"
              style={vars({ "--step": 2 })}
            >
              {t.why.body}
            </p>
          </div>
        </div>
        <EditorialPhoto
          slot="careAtHome"
          className="mt-16 aspect-[4/3] sm:aspect-[16/9] lg:mt-24 lg:aspect-[21/9]"
        />
      </div>
    </section>
  );
}

/**
 * The journey as one connected path, not three cards: describe, understand
 * what to do first, connect. Beside it, the only product UI on the page.
 */
function HowItWorks() {
  const { t } = useCopy();
  return (
    <section id="how" aria-labelledby="how-title" className="scroll-mt-16">
      <div className={`${CONTAINER} ${SECTION_Y}`}>
        <h2 id="how-title" className="font-display text-headline">
          <Lines lines={t.how.title} />
        </h2>

        <div className="mt-16 grid gap-16 lg:mt-24 lg:grid-cols-12 lg:gap-6">
          {/* The signature moment: as the list scrolls through the reading
              zone, StringProgress writes --progress (0 to 1) here, the
              forest path grows down the Setu line, and each stage comes
              forward as the path reaches it (see "care journey" in
              globals.css). */}
          {/* The reading line sits 40% above the bottom of the screen:
              progress is 0 when the list's top reaches it and 1 when the
              list's bottom does, so the path's tip is wherever you are
              reading. (In StringTune, offset-bottom moves the start and
              offset-top the end.) */}
          <ol
            className="relative lg:col-span-6"
            data-string="progress"
            data-string-id="care-journey-progress"
            data-string-enter-el="top"
            data-string-enter-vp="bottom"
            data-string-exit-el="bottom"
            data-string-exit-vp="bottom"
            data-string-offset-bottom="-40%"
            data-string-offset-top="40%"
          >
            {/* The Setu line joining the three stages, and the path that
                travels along it. */}
            <span aria-hidden className="absolute bottom-4 left-[3px] top-4 w-px bg-fresh" />
            <span
              aria-hidden
              className="journey-path absolute bottom-4 left-[3px] top-4 w-px bg-primary"
            />
            {t.how.steps.map((step, i) => (
              <li
                key={step.title}
                className="journey-stage relative pb-14 pl-10 last:pb-0 sm:pl-12"
                style={vars({ "--at": STAGE_AT[i] })}
              >
                <span
                  aria-hidden
                  className="journey-dot absolute left-0 top-[0.7em] size-[7px] rounded-full bg-primary"
                />
                <h3 className="font-display text-title">{step.title}</h3>
                <p className="mt-2 max-w-[42ch] leading-relaxed text-muted-foreground">
                  {step.body}
                </p>
              </li>
            ))}
          </ol>

          {/* Revealed once, in order: the frame, the question, the answer,
              then the way in. */}
          <div
            data-reveal="group"
            className="lg:col-span-5 lg:col-start-8"
            style={vars({ "--stagger": "130ms" })}
          >
            <div data-reveal-step className="rounded-[12px] border bg-card p-6 sm:p-8">
              <ExampleExchange showNext={false} staged />
            </div>
            <div data-reveal-step style={vars({ "--step": 3 })}>
              <Link
                href="/patient/"
                className="group mt-6 inline-flex min-h-11 items-center gap-2 font-semibold text-primary-text underline-offset-4 hover:underline"
              >
                {t.how.cta}
                <NudgeArrow />
              </Link>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

/** Who may answer a patient, as an editorial list rather than cards. */
function Professionals() {
  const { t } = useCopy();
  return (
    <section id="professionals" aria-labelledby="pros-title" className="scroll-mt-16 border-t">
      <div className={`${CONTAINER} ${SECTION_Y} grid gap-14 lg:grid-cols-12 lg:gap-6`}>
        <div data-reveal className="lg:col-span-6" style={vars({ "--reveal-y": "20px" })}>
          <h2 id="pros-title" className="font-display text-headline">
            <Lines lines={t.pros.title} />
          </h2>
          <p className="mt-8 max-w-[40ch] leading-relaxed text-muted-foreground">{t.pros.body}</p>
          <Link
            href="/doctor/"
            className="group mt-6 inline-flex min-h-11 items-center gap-2 font-semibold text-primary-text underline-offset-4 hover:underline"
          >
            {t.pros.apply}
            <NudgeArrow />
          </Link>
        </div>

        {/* Each profession in turn, each with its rule drawing in from the
            left. Once. */}
        <div data-reveal="group" className="lg:col-span-5 lg:col-start-8 lg:pt-4">
          <dl className="reveal-rules">
            {t.pros.categories.map(({ who, needs }, i) => (
              <div
                key={who}
                data-reveal-step
                className="grid gap-1 py-6 sm:grid-cols-[11.5rem_minmax(0,1fr)] sm:gap-6 sm:py-8"
                style={vars({ "--step": i + 1 })}
              >
                <dt className="font-display text-title">{who}</dt>
                <dd className="leading-relaxed text-muted-foreground sm:pt-1">{needs}</dd>
              </div>
            ))}
          </dl>
          <p
            data-reveal-step
            className="mt-6 text-sm text-muted-foreground"
            style={vars({ "--step": t.pros.categories.length + 1 })}
          >
            {t.pros.everyone}
          </p>
        </div>
      </div>
    </section>
  );
}

/** One statement of what SwasthyaSetu is, and is not. The one dark band. */
function Safety() {
  const { t } = useCopy();
  return (
    <section
      id="safety"
      aria-labelledby="safety-title"
      className="scroll-mt-16 bg-deep text-deep-foreground"
    >
      {/* One composition, one quiet reveal. */}
      <div
        data-reveal
        className={`${CONTAINER} ${SECTION_Y} grid gap-10 lg:grid-cols-12 lg:gap-6`}
      >
        <h2 id="safety-title" className="font-display text-headline lg:col-span-6">
          {t.safety.title}
        </h2>
        <p className="text-lede text-deep-foreground/85 lg:col-span-5 lg:col-start-8 lg:pt-4">
          {t.safety.body}
        </p>
      </div>
    </section>
  );
}

function Footer() {
  const { t } = useCopy();
  const link =
    "inline-flex min-h-11 items-center text-muted-foreground transition-colors duration-150 hover:text-foreground sm:min-h-0";
  return (
    <footer className={`${CONTAINER} grid gap-8 py-12 lg:grid-cols-12 lg:gap-6`}>
      <div className="lg:col-span-4">
        <BrandLockup />
        <p className="mt-3 text-sm text-muted-foreground" translate="no">
          Bridging Health. Delivering Hope.
        </p>
      </div>
      {/* Only what the header doesn't already offer. */}
      <nav aria-label="Footer" className="lg:col-span-4">
        <ul className="flex flex-wrap gap-x-6 gap-y-1 text-sm">
          <li><Link href="/doctor/" className={link}>{t.footer.forProfessionals}</Link></li>
          <li><Link href="/account/" className={link}>{t.footer.account}</Link></li>
        </ul>
      </nav>
      <p className="max-w-[40ch] text-sm text-muted-foreground lg:col-span-4">
        {t.footer.disclaimer}
      </p>
    </footer>
  );
}

/**
 * The landing page: a healthcare brand first, a product second. Every way
 * to get help opens the first-aid chat at /patient/ (no sign-in), and 102
 * is always in the header and the hero.
 */
export function HomeView() {
  return (
    <>
      <Hero />
      <Why />
      <HowItWorks />
      <Professionals />
      <Safety />
      <Footer />
      {/* Motion: one StringTune runtime (desktop), one reveal observer,
          and the hero's once-per-load entrance. None of it is needed to
          read or use the page. */}
      <StringTuneInit />
      <RevealObserver />
      <HeroIntro />
    </>
  );
}
