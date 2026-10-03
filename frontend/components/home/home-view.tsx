"use client";

import Link from "next/link";
import {
  ArrowRight,
  BadgeCheck,
  EyeOff,
  GraduationCap,
  Languages,
  MapPin,
  Phone,
  Pill,
  ShieldCheck,
  Siren,
  Stethoscope,
} from "lucide-react";

import { BrandLockup } from "@/components/brand";
import { SetuPath } from "@/components/home/setu-path";
import { PatientView } from "@/components/patient/patient-view";
import { Button } from "@/components/ui/button";
import { useCopy } from "@/lib/i18n";

const CONTAINER = "mx-auto w-full max-w-[1240px] px-4 sm:px-6";
const SECTION_Y = "py-20 sm:py-24 lg:py-28";

/** Bring the live chat into view and put the cursor in it. */
function startFirstAid() {
  const reduce = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
  document
    .getElementById("first-aid")
    ?.scrollIntoView({ behavior: reduce ? "auto" : "smooth", block: "start" });
  window.setTimeout(() => {
    document
      .querySelector<HTMLInputElement>("#first-aid input[name=message]")
      ?.focus({ preventScroll: true });
  }, reduce ? 0 : 350);
}

function Hero({ chat, talk }: { chat: React.ReactNode; talk: () => void }) {
  const { t } = useCopy();
  const trustIcons = [Languages, BadgeCheck, MapPin];
  return (
    <section
      aria-labelledby="hero-title"
      className={`${CONTAINER} grid gap-12 pb-20 pt-10 sm:pt-14 lg:grid-cols-[minmax(0,1fr)_minmax(0,440px)] lg:items-center lg:gap-12 xl:grid-cols-[minmax(0,1fr)_minmax(0,520px)] xl:gap-16 lg:pb-24 lg:pt-16`}
    >
      <div className="flex flex-col">
        <h1
          id="hero-title"
          className="text-display font-extrabold text-foreground animate-in fade-in slide-in-from-bottom-2 duration-500"
        >
          <span className="block">{t.hero.titleA}</span>
          <span className="block text-primary-text">{t.hero.titleB}</span>
        </h1>
        <p className="mt-6 max-w-[34rem] text-lede text-muted-foreground">{t.hero.lede}</p>

        <div className="mt-8 flex flex-col gap-3 sm:flex-row">
          <Button size="lg" onClick={startFirstAid}>
            {t.hero.primary}
          </Button>
          <Button size="lg" variant="outline" onClick={talk}>
            {t.hero.secondary}
          </Button>
        </div>

        {/* Emergency: separate from the product actions, and the only red. */}
        <div className="mt-8 flex flex-wrap items-center justify-between gap-3 rounded-[14px] border border-triage-red/25 bg-triage-red/[0.05] px-4 py-3 sm:max-w-[34rem]">
          <p className="flex items-center gap-2.5 font-semibold text-foreground">
            <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-triage-red/10 text-triage-red">
              <Phone aria-hidden className="size-4" />
            </span>
            {t.hero.emergencyQ}
          </p>
          <Button asChild variant="emergency" size="sm" className="tap-target sm:h-10">
            <a href="tel:102">{t.hero.emergencyCall}</a>
          </Button>
        </div>

        <ul className="mt-8 flex flex-wrap gap-x-6 gap-y-3 text-sm font-medium text-muted-foreground">
          {t.hero.trust.map((item, i) => {
            const Icon = trustIcons[i];
            return (
              <li key={item} className="flex items-center gap-2">
                <Icon aria-hidden className="size-4 text-primary-text" />
                {item}
              </li>
            );
          })}
        </ul>
      </div>

      <div className="h-[78dvh] min-h-[560px] lg:h-[min(660px,calc(100dvh-8rem))]">{chat}</div>
    </section>
  );
}

function HowItWorks() {
  const { t } = useCopy();
  return (
    <section id="how" aria-labelledby="how-title" className="scroll-mt-16 border-y bg-card">
      <div className={`${CONTAINER} ${SECTION_Y}`}>
        <h2 id="how-title" className="max-w-xl text-headline font-extrabold">
          {t.how.title}
        </h2>
        <ol className="relative mt-14 grid gap-12 md:mt-16 md:grid-cols-3 md:gap-10">
          <SetuPath
            orientation="horizontal"
            className="absolute left-5 top-[10px] hidden h-6 w-[calc(100%-1.25rem)] md:block"
          />
          <SetuPath
            orientation="vertical"
            className="absolute left-[10px] top-6 h-[calc(100%-3rem)] w-6 md:hidden"
          />
          {t.how.steps.map((step, i) => (
            <li key={step.title} className="relative pl-16 md:pl-0 md:pt-16">
              <span className="absolute left-0 top-0 flex size-11 items-center justify-center rounded-full border-2 border-primary bg-card font-heading text-sm font-bold text-primary-text tabular">
                {String(i + 1).padStart(2, "0")}
              </span>
              <h3 className="text-title font-bold">{step.title}</h3>
              <p className="mt-2 max-w-[32ch] leading-relaxed text-muted-foreground">{step.body}</p>
            </li>
          ))}
        </ol>
      </div>
    </section>
  );
}

function Professionals() {
  const { t } = useCopy();
  const icons = [Stethoscope, Pill, GraduationCap];
  return (
    <section id="professionals" aria-labelledby="pros-title" className={`scroll-mt-16 ${CONTAINER} ${SECTION_Y}`}>
      <div className="grid overflow-hidden rounded-[20px] border bg-card lg:grid-cols-[minmax(0,5fr)_minmax(0,7fr)]">
        <div className="flex flex-col bg-soft p-8 sm:p-10 lg:p-12">
          <h2
            id="pros-title"
            className="font-heading text-[1.75rem] font-extrabold leading-[1.15] tracking-[-0.02em] sm:text-[2rem]"
          >
            {t.pros.title}
          </h2>
          <p className="mt-5 max-w-[46ch] leading-relaxed text-foreground/80">{t.pros.body}</p>

          <ol className="relative mt-8 flex flex-col gap-4 pl-8">
            <span aria-hidden className="absolute bottom-2 left-[7px] top-2 w-px bg-primary/40" />
            {t.pros.process.map((step) => (
              <li key={step} className="relative font-semibold">
                <span
                  aria-hidden
                  className="absolute -left-8 top-[5px] size-[15px] rounded-full border-2 border-primary bg-soft"
                />
                {step}
              </li>
            ))}
          </ol>

          <div className="mt-10 lg:mt-auto lg:pt-10">
            <p className="text-sm text-foreground/75">{t.pros.badgeCaption}</p>
            <span className="mt-2 inline-flex items-center gap-2 rounded-[10px] border border-[#a8d8c7]/30 bg-deep px-3 py-2 text-sm font-semibold text-deep-foreground">
              <BadgeCheck aria-hidden className="size-4 text-[#a8d8c7]" />
              Verified Doctor
            </span>
          </div>
        </div>

        <div className="p-8 sm:p-10 lg:p-12">
          <h3 className="text-sm font-semibold text-muted-foreground">{t.pros.requiresTitle}</h3>
          <dl className="mt-4 divide-y">
            {t.pros.categories.map(({ who, needs }, i) => {
              const Icon = icons[i];
              return (
                <div key={who} className="grid gap-1.5 py-5 sm:grid-cols-[11rem_minmax(0,1fr)] sm:gap-6">
                  <dt className="flex items-center gap-3 font-heading font-bold">
                    <Icon aria-hidden className="size-5 text-primary-text" />
                    {who}
                  </dt>
                  <dd className="leading-relaxed text-muted-foreground">{needs}</dd>
                </div>
              );
            })}
          </dl>
          <p className="border-t pt-5 text-sm text-muted-foreground">{t.pros.everyone}</p>
          <Link
            href="/doctor/"
            className="mt-8 inline-flex min-h-11 items-center gap-2 font-semibold text-primary-text underline-offset-4 hover:underline"
          >
            {t.pros.apply}
            <ArrowRight aria-hidden className="size-4" />
          </Link>
        </div>
      </div>
    </section>
  );
}

function HumanAndSafety() {
  const { t } = useCopy();
  const icons = [ShieldCheck, Siren, EyeOff, BadgeCheck];
  return (
    <section id="safety" aria-labelledby="human-title" className="scroll-mt-16 bg-deep text-deep-foreground">
      <div className={`${CONTAINER} ${SECTION_Y} grid gap-14 lg:grid-cols-2 lg:gap-20`}>
        <div>
          <h2 id="human-title" className="text-headline font-extrabold">
            {t.human.lines.map((line) => (
              <span key={line} className="block">
                {line}
              </span>
            ))}
          </h2>
          {/* The band is dark in both themes, so it always uses the light
              accent (7.6:1 on the deep teal). */}
          <p className="mt-8 max-w-[30ch] text-lede font-semibold text-[#a8d8c7]">
            {t.human.answer}
          </p>
        </div>
        <div>
          <h3 className="text-title font-bold">{t.human.safetyTitle}</h3>
          <ul className="mt-4 divide-y divide-white/10">
            {t.human.items.map(({ title, body }, i) => {
              const Icon = icons[i];
              return (
                <li key={title} className="flex gap-4 py-5">
                  <Icon aria-hidden className="mt-0.5 size-5 shrink-0 text-[#a8d8c7]" />
                  <div>
                    <p className="font-semibold">{title}</p>
                    <p className="mt-1 leading-relaxed text-deep-foreground/75">{body}</p>
                  </div>
                </li>
              );
            })}
          </ul>
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
    <footer className="border-t">
      <div className={`${CONTAINER} grid gap-10 py-12 md:grid-cols-[1.2fr_1fr_1.4fr]`}>
        <div>
          <BrandLockup />
          <p className="mt-3 text-sm text-muted-foreground" translate="no">
            Bridging Health. Delivering Hope.
          </p>
        </div>
        <nav aria-label="Footer">
          <ul className="grid gap-1 text-sm sm:gap-2">
            <li><a href="#how" className={link}>{t.nav.how}</a></li>
            <li><a href="#professionals" className={link}>{t.nav.professionals}</a></li>
            <li><a href="#safety" className={link}>{t.nav.safety}</a></li>
            <li><Link href="/doctor/" className={link}>{t.pros.apply}</Link></li>
            <li><Link href="/account/" className={link}>{t.footer.account}</Link></li>
          </ul>
        </nav>
        <div className="text-sm leading-relaxed text-muted-foreground">
          <p>{t.footer.disclaimer}</p>
          <p className="mt-2 font-semibold text-foreground">
            {t.hero.emergencyQ}{" "}
            <a href="tel:102" className="text-triage-red underline-offset-4 hover:underline">
              {t.hero.emergencyCall}
            </a>
          </p>
        </div>
      </div>
    </footer>
  );
}

/**
 * The landing page. Emergency first: the live first-aid chat sits in the
 * hero, usable immediately with no account, and 102 is always in view.
 * Requesting a professional and the call itself take over the page.
 */
export function HomeView() {
  return (
    <PatientView
      renderChat={(chat, talk) => (
        <>
          <Hero chat={chat} talk={talk} />
          <HowItWorks />
          <Professionals />
          <HumanAndSafety />
          <Footer />
        </>
      )}
    />
  );
}
