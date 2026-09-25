import { Activity, HeartPulse, Stethoscope, WifiOff } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

/**
 * Week 1 placeholder home page. The real patient / doctor / admin views
 * arrive with the frontend foundations work in Week 6; this page exists to
 * prove the stack renders end to end.
 */

const capabilities = [
  {
    icon: HeartPulse,
    title: "AI-assisted triage",
    description:
      "Describe symptoms in Nepali by voice or text. A six-stage pipeline extracts them, classifies urgency, and grounds its answer in curated first-aid sources.",
  },
  {
    icon: Stethoscope,
    title: "Volunteer doctors",
    description:
      "Ranked by location, specialization, availability, language, and experience — then connected over a peer-to-peer video call.",
  },
  {
    icon: WifiOff,
    title: "Works offline",
    description:
      "Cached first-aid guidance, emergency contacts, and GPS stay available with no signal. Reports logged offline sync on reconnect.",
  },
];

export default function Home() {
  return (
    <main className="mx-auto flex min-h-dvh max-w-3xl flex-col justify-center gap-10 px-6 py-16">
      <header className="flex flex-col gap-3">
        <p className="text-sm font-medium tracking-wide text-primary">
          स्वास्थ्य सेतु
        </p>
        <h1 className="text-4xl font-semibold tracking-tight sm:text-5xl">
          SwasthyaSetu
        </h1>
        <p className="text-lg text-muted-foreground">The bridge to health.</p>
        <p className="max-w-prose text-balance text-muted-foreground">
          A healthcare access platform for communities where qualified medical
          help is far away, unreliable, or out of reach.
        </p>
      </header>

      <section className="grid gap-4 sm:grid-cols-3" aria-label="What it does">
        {capabilities.map(({ icon: Icon, title, description }) => (
          <Card key={title}>
            <CardHeader>
              <Icon aria-hidden className="size-5 text-primary" />
              <CardTitle>{title}</CardTitle>
              <CardDescription>{description}</CardDescription>
            </CardHeader>
          </Card>
        ))}
      </section>

      <Card className="border-dashed">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-sm font-medium">
            <Activity aria-hidden className="size-4 text-primary" />
            Week 1 skeleton
          </CardTitle>
          <CardDescription>
            Next.js PWA, FastAPI backend, PostgreSQL, Redis, and Celery are
            wired and running. Patient, doctor, and admin views arrive in
            Week 6.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-3">
          <Button asChild variant="outline" size="sm">
            <a href="http://localhost:8000/docs">API docs</a>
          </Button>
          <Button asChild variant="ghost" size="sm">
            <a href="http://localhost:8000/health/ready">Service health</a>
          </Button>
        </CardContent>
      </Card>
    </main>
  );
}
