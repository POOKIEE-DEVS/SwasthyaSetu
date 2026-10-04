"use client";

import Link from "next/link";
import { AlertCircle, BadgeCheck, Clock, Loader2, RefreshCw, Stethoscope } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ROLE_LABEL } from "@/lib/api";
import { isProfessional, useAuth, useAuthStore } from "@/lib/store/auth";

function Notice({
  icon,
  title,
  children,
  actions,
}: {
  icon: React.ReactNode;
  title: string;
  children: React.ReactNode;
  actions: React.ReactNode;
}) {
  return (
    <div className="mx-auto w-full max-w-md px-4 py-16">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-xl">
            {icon}
            {title}
          </CardTitle>
          <CardDescription>{children}</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">{actions}</CardContent>
      </Card>
    </div>
  );
}

/**
 * The professional area is for verified doctors, pharmacists, nurses,
 * paramedics and MBBS students only. Everyone else sees where they stand and what to do next.
 * The server enforces the same rule on every request; this is the
 * friendly version of it.
 */
export function ProfessionalGate({ children }: { children: React.ReactNode }) {
  const { user, ready, verifiedRole } = useAuth();

  if (!ready) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 aria-hidden className="size-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (verifiedRole && user) {
    return (
      <>
        <div className="border-b bg-triage-green/10">
          <p className="mx-auto flex max-w-6xl items-center gap-2 px-4 py-2 text-sm">
            <BadgeCheck aria-hidden className="size-4 text-triage-green" />
            Verified {ROLE_LABEL[verifiedRole]} · {user.verification?.full_name ?? user.name}
          </p>
        </div>
        {children}
      </>
    );
  }

  if (!user) {
    return (
      <Notice
        icon={<Stethoscope aria-hidden className="size-5 text-primary-text" />}
        title="For medical professionals"
        actions={
          <Button asChild size="lg">
            <Link href="/account/?next=/doctor/">Sign in</Link>
          </Button>
        }
      >
        Verified doctors, pharmacists, nurses, paramedics and MBBS students take video calls
        from patients here.
        Sign in to apply, or to go online if you are already verified.
      </Notice>
    );
  }

  const verification = user.verification;
  const refresh = (
    <Button variant="outline" onClick={() => void useAuthStore.getState().load()}>
      <RefreshCw aria-hidden />
      Check again
    </Button>
  );

  if (!isProfessional(user.role) || !verification) {
    return (
      <Notice
        icon={<Stethoscope aria-hidden className="size-5 text-primary-text" />}
        title="Get verified to take calls"
        actions={
          <Button asChild size="lg">
            <Link href="/apply/">Apply for verification</Link>
          </Button>
        }
      >
        This area is for doctors, pharmacists, nurses, paramedics and MBBS students whose
        documents have been checked. It takes a few minutes to apply.
      </Notice>
    );
  }

  if (verification.status === "rejected") {
    return (
      <Notice
        icon={<AlertCircle aria-hidden className="size-5 text-destructive" />}
        title="Your application needs changes"
        actions={
          <Button asChild size="lg">
            <Link href="/apply/">Fix and resubmit</Link>
          </Button>
        }
      >
        <span className="block">The reviewer said: {verification.rejection_reason}</span>
      </Notice>
    );
  }

  return (
    <Notice
      icon={<Clock aria-hidden className="size-5 text-triage-yellow" />}
      title="Waiting for review"
      actions={
        <>
          {refresh}
          <Button asChild variant="ghost">
            <Link href="/apply/">View or update my application</Link>
          </Button>
        </>
      }
    >
      Thanks, {verification.full_name}. The admin is checking your{" "}
      {ROLE_LABEL[verification.role]} documents. You can take patient calls as soon as you
      are approved.
    </Notice>
  );
}
