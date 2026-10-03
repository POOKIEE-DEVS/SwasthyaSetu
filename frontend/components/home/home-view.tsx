"use client";

import Link from "next/link";
import { ArrowRight, BadgeCheck, MessageCircle, Phone, Stethoscope, Video } from "lucide-react";

import { BrandLockup } from "@/components/brand";
import { Testimonials } from "@/components/home/testimonials";
import { PatientView } from "@/components/patient/patient-view";
import { Button } from "@/components/ui/button";

const STEPS = [
  {
    icon: MessageCircle,
    title: "Describe it",
    body: "Type what happened, in English or Nepali. No account and no forms.",
  },
  {
    icon: Stethoscope,
    title: "Get first-aid steps",
    body: "MedGemma, a medical AI model, replies with short, practical steps. Emergencies show 102 straight away.",
  },
  {
    icon: Video,
    title: "See a verified doctor",
    body: "Request a call. A checked professional joins you on video, with your chat already shared.",
  },
];

const PROFESSIONALS = [
  { who: "Doctors", check: "Nepal Medical Council registration and certificate" },
  { who: "Pharmacists", check: "Nepal Pharmacy Council registration and certificate" },
  { who: "MBBS students", check: "A letter of recommendation from a registered doctor" },
];

function Hero({ chat }: { chat: React.ReactNode }) {
  return (
    <section className="mx-auto grid w-full max-w-6xl items-center gap-8 px-4 pb-12 pt-8 lg:min-h-[calc(100dvh-4rem)] lg:grid-cols-[minmax(0,1fr)_minmax(0,500px)] lg:gap-12 lg:pb-16 lg:pt-10">
      <div className="flex flex-col items-start gap-5 animate-in fade-in slide-in-from-bottom-3 duration-700">
        <p className="text-sm font-bold text-primary-text" lang="ne">
          स्वास्थ्य सेतु
        </p>
        <h1 className="text-[2.15rem] font-bold leading-[1.08] tracking-tight sm:text-5xl lg:text-[3.35rem]">
          First aid now.
          <br />
          A verified doctor next.
        </h1>
        <p className="max-w-[46ch] text-base leading-relaxed text-muted-foreground sm:text-lg">
          Describe what happened in English or Nepali. Get first-aid steps, then talk to a
          verified doctor on video.
        </p>
        <Button asChild variant="emergency" size="lg" className="tap-target">
          <a href="tel:102">
            <Phone aria-hidden />
            Life in danger? Call 102
          </a>
        </Button>
      </div>
      <div className="h-[72dvh] min-h-[520px] animate-in fade-in slide-in-from-bottom-4 duration-700 lg:h-[min(680px,calc(100dvh-7rem))]">
        {chat}
      </div>
    </section>
  );
}

function Problem() {
  return (
    <section aria-labelledby="problem-title" className="border-t bg-muted/60">
      <div className="mx-auto max-w-4xl px-4 py-20">
        <h2
          id="problem-title"
          className="text-3xl font-bold leading-tight tracking-tight sm:text-[2.6rem]"
        >
          When someone gets hurt far from a clinic, families decide what to do alone.
        </h2>
        <p className="mt-5 max-w-[60ch] text-lg leading-relaxed text-muted-foreground">
          Help can be hours away, and the first minutes matter most. People need clear
          steps in their own language, and a real doctor as soon as possible.
        </p>
      </div>
    </section>
  );
}

function HowItWorks() {
  return (
    <section aria-labelledby="how-title" className="border-t">
      <div className="mx-auto grid max-w-6xl gap-10 px-4 py-20 lg:grid-cols-[minmax(0,5fr)_minmax(0,7fr)] lg:gap-16">
        <div className="lg:sticky lg:top-28 lg:self-start">
          <h2 id="how-title" className="text-3xl font-bold tracking-tight sm:text-4xl">
            Help in three moves, in your language.
          </h2>
          <p className="mt-4 max-w-[42ch] text-muted-foreground">
            Built for the moment something goes wrong, when the nearest clinic is hours away.
          </p>
        </div>
        <ol className="relative flex flex-col gap-10 before:absolute before:bottom-6 before:left-5 before:top-6 before:w-px before:bg-border">
          {STEPS.map(({ icon: Icon, title, body }) => (
            <li key={title} className="relative flex gap-5">
              <span className="relative flex size-11 shrink-0 items-center justify-center rounded-full bg-card text-primary-text shadow-raised-sm">
                <Icon aria-hidden className="size-[18px]" />
              </span>
              <div className="pt-1.5">
                <h3 className="text-lg font-bold">{title}</h3>
                <p className="mt-1.5 max-w-[52ch] leading-relaxed text-muted-foreground">{body}</p>
              </div>
            </li>
          ))}
        </ol>
      </div>
    </section>
  );
}

function WhoAnswers() {
  return (
    <section aria-labelledby="who-title" className="mx-auto max-w-6xl px-4 py-20">
      <div className="grid gap-4 lg:grid-cols-5 lg:grid-rows-3">
        <div className="relative isolate flex flex-col justify-between gap-10 overflow-hidden rounded-[20px] bg-[#164e63] p-8 text-white shadow-raised lg:col-span-3 lg:row-span-3 lg:p-10">
          {/* The logo's ribbon, large and faint, as the tile's texture. */}
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src="/brand/mark.png"
            alt=""
            width={512}
            height={512}
            loading="lazy"
            className="pointer-events-none absolute -bottom-24 -right-20 -z-10 size-80 opacity-[0.12] brightness-[3] grayscale"
          />
          <div>
            <h2 id="who-title" className="max-w-[16ch] text-3xl font-bold tracking-tight sm:text-4xl">
              Every professional is checked by hand.
            </h2>
            <p className="mt-4 max-w-[46ch] leading-relaxed text-white/85">
              Before anyone can see a patient, an admin reviews their citizenship and their
              registration. Patients always see who is on the call.
            </p>
          </div>
          <span className="flex w-fit items-center gap-2 rounded-full bg-white/10 px-3.5 py-2 text-sm font-medium ring-1 ring-white/15">
            <BadgeCheck aria-hidden className="size-4 text-[#6ee7b7]" />
            Verified Doctor · shown on every call
          </span>
        </div>
        {PROFESSIONALS.map(({ who, check }, i) => (
          <div
            key={who}
            className={
              i === 0
                ? "rounded-[20px] border border-cta/30 bg-cta/10 p-6 lg:col-span-2"
                : "rounded-[20px] bg-card p-6 shadow-raised lg:col-span-2"
            }
          >
            <h3 className="font-bold">{who}</h3>
            <p className="mt-1 text-sm leading-relaxed text-muted-foreground">{check}</p>
          </div>
        ))}
      </div>
    </section>
  );
}

function Closing() {
  return (
    <section className="border-t">
      <div className="mx-auto flex max-w-3xl flex-col items-center gap-6 px-4 py-24 text-center">
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src="/brand/mark.png"
          alt=""
          width={512}
          height={512}
          loading="lazy"
          className="size-20 dark:rounded-full dark:bg-[oklch(0.96_0.01_255)] dark:p-2"
        />
        <p className="text-3xl font-bold leading-[1.1] tracking-tight sm:text-5xl">
          <span className="block">Bridging Health.</span>
          <span className="block text-primary-text">Delivering Hope.</span>
        </p>
        <p className="max-w-[48ch] text-muted-foreground">
          Doctors, pharmacists and MBBS students can volunteer their time once their documents
          are checked.
        </p>
        <Button asChild variant="cta" size="lg">
          <Link href="/doctor/">
            Join as a professional
            <ArrowRight aria-hidden />
          </Link>
        </Button>
      </div>
    </section>
  );
}

function Footer() {
  return (
    <footer className="border-t bg-muted/60">
      <div className="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-10 sm:flex-row sm:items-center sm:justify-between">
        <BrandLockup />
        <p className="max-w-[52ch] text-sm text-muted-foreground">
          First-aid information, not a medical diagnosis. In an emergency in Nepal, call{" "}
          <a href="tel:102" className="font-semibold text-triage-red underline-offset-4 hover:underline">
            102
          </a>
          .
        </p>
      </div>
    </footer>
  );
}

/**
 * The home page. Emergency first: the live first-aid chat is in the hero,
 * usable immediately with no account. Requesting a doctor and the call take
 * over the whole page, without the marketing sections around them.
 */
export function HomeView() {
  return (
    <PatientView
      renderChat={(chat) => (
        <>
          <Hero chat={chat} />
          <Problem />
          <HowItWorks />
          <WhoAnswers />
          <Testimonials />
          <Closing />
          <Footer />
        </>
      )}
    />
  );
}
