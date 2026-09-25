import Link from "next/link";
import { HeartPulse, Stethoscope } from "lucide-react";

import { SiteHeader } from "@/components/site-header";

const roles = [
  {
    href: "/patient/",
    icon: HeartPulse,
    title: "I need help",
    nepali: "मलाई सहयोग चाहियो",
    description: "Get first-aid guidance from the AI assistant, then talk to a volunteer doctor.",
  },
  {
    href: "/doctor/",
    icon: Stethoscope,
    title: "I'm a doctor",
    nepali: "म डाक्टर हुँ",
    description: "Go online, see waiting patients, and take their video calls.",
  },
];

export default function Home() {
  return (
    <>
      <SiteHeader />
      <main className="mx-auto flex w-full max-w-3xl flex-col gap-10 px-4 py-12 sm:py-20">
        <div className="flex flex-col gap-3 text-center">
          <p className="text-sm font-medium tracking-wide text-primary">स्वास्थ्य सेतु</p>
          <h1 className="text-4xl font-semibold tracking-tight sm:text-5xl">The bridge to health</h1>
          <p className="mx-auto max-w-xl text-balance text-muted-foreground">
            First-aid guidance and a live doctor on video, for places where medical help is far
            away.
          </p>
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          {roles.map(({ href, icon: Icon, title, nepali, description }) => (
            <Link
              key={href}
              href={href}
              className="group flex flex-col gap-3 rounded-xl border bg-card p-6 transition-colors hover:border-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <Icon aria-hidden className="size-8 text-primary" />
              <div>
                <h2 className="text-xl font-semibold">{title}</h2>
                <p className="text-sm text-muted-foreground">{nepali}</p>
              </div>
              <p className="text-sm text-muted-foreground">{description}</p>
            </Link>
          ))}
        </div>

        <p className="text-center text-xs text-muted-foreground">
          SwasthyaSetu gives first-aid information, not a medical diagnosis. In an emergency,
          call 102.
        </p>
      </main>
    </>
  );
}
