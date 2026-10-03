"use client";

import Link from "next/link";
import { ArrowRight, Phone } from "lucide-react";

import { BrandLockup } from "@/components/brand";
import { ExampleExchange } from "@/components/chat/example-exchange";
import { EditorialPhoto } from "@/components/home/editorial-photo";
import { SetuBridge } from "@/components/home/setu-bridge";
import { Button } from "@/components/ui/button";
import { useCopy } from "@/lib/i18n";
import { cn } from "@/lib/utils";

const CONTAINER = "mx-auto w-full max-w-[1240px] px-4 sm:px-6";
/** Sections breathe: 96px on phones, up to 160px on large screens. */
const SECTION_Y = "py-24 lg:py-32 xl:py-40";

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
    <Button asChild size="lg" className="h-12 px-6">
      <Link href="/patient/">
        {label}
        <ArrowRight aria-hidden />
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
        "inline-flex min-h-12 items-center justify-center gap-2.5 rounded-[8px] border border-input px-5 text-[15px] text-foreground transition-colors duration-150 hover:border-foreground/40 sm:justify-start sm:border-0 sm:px-0",
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
      className={`${CONTAINER} pb-20 pt-14 sm:pt-20 lg:pb-24 lg:pt-28`}
    >
      <h1 id="hero-title" className="font-display text-display">
        <Lines lines={t.hero.title} />
      </h1>

      <div className="mt-8 lg:mt-14 lg:grid lg:grid-cols-12 lg:gap-6">
        <div className="lg:col-span-6 lg:col-start-7 xl:col-span-5 xl:col-start-7">
          <p className="max-w-[36ch] text-lede text-muted-foreground">{t.hero.lede}</p>
          <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:items-center sm:gap-8">
            <GetHelpButton label={t.hero.cta} />
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
        <div className="grid gap-6 lg:grid-cols-12">
          <p className="max-w-[24ch] font-medium text-muted-foreground lg:col-span-3 lg:pt-5">
            {t.why.lead}
          </p>
          <div className="lg:col-span-8 lg:col-start-5">
            <h2 id="why-title" className="font-display text-headline">
              {t.why.title}
            </h2>
            <p className="mt-8 max-w-[44ch] text-lede text-foreground/80">{t.why.body}</p>
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
          <ol className="relative lg:col-span-6">
            {/* The Setu line joining the three stages. */}
            <span aria-hidden className="absolute bottom-4 left-[3px] top-4 w-px bg-fresh" />
            {t.how.steps.map((step) => (
              <li key={step.title} className="relative pb-14 pl-10 last:pb-0 sm:pl-12">
                <span
                  aria-hidden
                  className="absolute left-0 top-[0.7em] size-[7px] rounded-full bg-primary"
                />
                <h3 className="font-display text-title">{step.title}</h3>
                <p className="mt-2 max-w-[42ch] leading-relaxed text-muted-foreground">
                  {step.body}
                </p>
              </li>
            ))}
          </ol>

          <div className="lg:col-span-5 lg:col-start-8">
            <div className="rounded-[12px] border bg-card p-6 sm:p-8">
              <ExampleExchange />
            </div>
            <Link
              href="/patient/"
              className="mt-6 inline-flex min-h-11 items-center gap-2 font-semibold text-primary-text underline-offset-4 hover:underline"
            >
              {t.how.cta}
              <ArrowRight aria-hidden className="size-4" />
            </Link>
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
        <div className="lg:col-span-6">
          <h2 id="pros-title" className="font-display text-headline">
            <Lines lines={t.pros.title} />
          </h2>
          <p className="mt-8 max-w-[40ch] leading-relaxed text-muted-foreground">{t.pros.body}</p>
          <Link
            href="/doctor/"
            className="mt-6 inline-flex min-h-11 items-center gap-2 font-semibold text-primary-text underline-offset-4 hover:underline"
          >
            {t.pros.apply}
            <ArrowRight aria-hidden className="size-4" />
          </Link>
        </div>

        <div className="lg:col-span-5 lg:col-start-8 lg:pt-4">
          <dl className="border-t border-foreground/70">
            {t.pros.categories.map(({ who, needs }) => (
              <div
                key={who}
                className="grid gap-1 border-b py-6 sm:grid-cols-[11.5rem_minmax(0,1fr)] sm:gap-6 sm:py-8"
              >
                <dt className="font-display text-title">{who}</dt>
                <dd className="leading-relaxed text-muted-foreground sm:pt-1">{needs}</dd>
              </div>
            ))}
          </dl>
          <p className="mt-6 text-sm text-muted-foreground">{t.pros.everyone}</p>
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
      <div className={`${CONTAINER} ${SECTION_Y} grid gap-10 lg:grid-cols-12 lg:gap-6`}>
        <h2 id="safety-title" className="font-display text-headline lg:col-span-6">
          {t.safety.title}
        </h2>
        <div className="lg:col-span-5 lg:col-start-8 lg:pt-4">
          <p className="text-lede text-deep-foreground/85">{t.safety.body}</p>
          <a
            href="tel:102"
            className="mt-8 inline-flex min-h-12 items-center gap-2.5 font-semibold underline decoration-deep-foreground/40 underline-offset-4 hover:decoration-deep-foreground"
          >
            <Phone aria-hidden className="size-4" />
            {t.hero.emergencyQ} {t.hero.emergencyCall}
          </a>
        </div>
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
      <nav aria-label="Footer" className="lg:col-span-4">
        <ul className="flex flex-wrap gap-x-6 gap-y-1 text-sm">
          <li><a href="#how" className={link}>{t.nav.how}</a></li>
          <li><a href="#professionals" className={link}>{t.nav.professionals}</a></li>
          <li><a href="#safety" className={link}>{t.nav.safety}</a></li>
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
    </>
  );
}
