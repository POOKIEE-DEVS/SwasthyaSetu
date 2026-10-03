"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  BadgeCheck,
  GraduationCap,
  HeartPulse,
  Loader2,
  LogOut,
  Pill,
  ShieldCheck,
  Stethoscope,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { api, ApiError, ROLE_LABEL, type Role } from "@/lib/api";
import { isProfessional, useAuth, useAuthStore } from "@/lib/store/auth";
import { useQueryParam } from "@/lib/use-query-param";
import { cn } from "@/lib/utils";

const ROLE_CHOICES: {
  role: Role;
  icon: typeof HeartPulse;
  nepali: string;
  description: string;
}[] = [
  {
    role: "patient",
    icon: HeartPulse,
    nepali: "बिरामी / सहयोग चाहिने",
    description: "Get first-aid help and keep your chats.",
  },
  {
    role: "doctor",
    icon: Stethoscope,
    nepali: "डाक्टर",
    description: "Needs your Nepal Medical Council number.",
  },
  {
    role: "pharmacist",
    icon: Pill,
    nepali: "फार्मासिस्ट",
    description: "Needs your Nepal Pharmacy Council number.",
  },
  {
    role: "student",
    icon: GraduationCap,
    nepali: "MBBS विद्यार्थी",
    description: "Needs a letter of recommendation from a doctor.",
  },
];

function safeNext(value: string | null): string {
  return value && value.startsWith("/") && !value.startsWith("//") ? value : "/";
}

function RolePicker({ onChosen }: { onChosen: (role: Role) => void }) {
  const [saving, setSaving] = useState<Role | null>(null);
  const [error, setError] = useState<string | null>(null);

  const choose = async (role: Role) => {
    setSaving(role);
    setError(null);
    try {
      await api.auth.chooseRole(role);
      await useAuthStore.getState().load();
      onChosen(role);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Couldn't save your choice.");
      setSaving(null);
    }
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="grid gap-3 sm:grid-cols-2">
        {ROLE_CHOICES.map(({ role, icon: Icon, nepali, description }) => (
          <button
            key={role}
            type="button"
            disabled={saving !== null}
            onClick={() => choose(role)}
            className={cn(
              "flex flex-col gap-2 rounded-xl border bg-card p-4 text-left transition-colors hover:border-primary disabled:opacity-60",
              saving === role && "border-primary",
            )}
          >
            <span className="flex items-center gap-2 font-semibold">
              {saving === role ? (
                <Loader2 aria-hidden className="size-5 animate-spin text-primary-text" />
              ) : (
                <Icon aria-hidden className="size-5 text-primary-text" />
              )}
              {ROLE_LABEL[role]}
            </span>
            <span className="text-sm text-muted-foreground">{nepali}</span>
            <span className="text-sm text-muted-foreground">{description}</span>
          </button>
        ))}
      </div>
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
    </div>
  );
}

function DevLoginForm({ onSignedIn }: { onSignedIn: () => void }) {
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  return (
    <form
      className="flex flex-col gap-2 rounded-lg border border-dashed p-3"
      onSubmit={async (event) => {
        event.preventDefault();
        setBusy(true);
        setError(null);
        try {
          await api.auth.devLogin(email, name || email.split("@")[0]);
          await useAuthStore.getState().load();
          onSignedIn();
        } catch (e) {
          setError(e instanceof ApiError ? e.message : "Sign-in failed.");
        } finally {
          setBusy(false);
        }
      }}
    >
      <p className="text-xs font-medium text-muted-foreground">
        Development sign-in (local testing only, off in production)
      </p>
      <Input
        type="email"
        required
        placeholder="email@example.com"
        aria-label="Development email"
        value={email}
        onChange={(event) => setEmail(event.target.value)}
      />
      <Input
        placeholder="Name"
        aria-label="Development name"
        value={name}
        onChange={(event) => setName(event.target.value)}
      />
      <Button type="submit" variant="outline" disabled={busy}>
        Sign in for testing
      </Button>
      {error && <p className="text-sm text-destructive">{error}</p>}
    </form>
  );
}

function GoogleIcon() {
  return (
    <svg aria-hidden viewBox="0 0 48 48" className="size-5">
      <path fill="#FFC107" d="M43.6 20.5H42V20H24v8h11.3C33.7 32.7 29.2 36 24 36c-6.6 0-12-5.4-12-12s5.4-12 12-12c3.1 0 5.8 1.2 7.9 3.1l5.7-5.7C34 6.1 29.3 4 24 4 12.9 4 4 12.9 4 24s8.9 20 20 20 20-8.9 20-20c0-1.3-.1-2.4-.4-3.5z" />
      <path fill="#FF3D00" d="m6.3 14.7 6.6 4.8C14.7 15.1 19 12 24 12c3.1 0 5.8 1.2 7.9 3.1l5.7-5.7C34 6.1 29.3 4 24 4 16.3 4 9.7 8.3 6.3 14.7z" />
      <path fill="#4CAF50" d="M24 44c5.2 0 9.9-2 13.4-5.2l-6.2-5.2C29.2 35.1 26.7 36 24 36c-5.2 0-9.6-3.3-11.3-8l-6.5 5C9.5 39.6 16.2 44 24 44z" />
      <path fill="#1976D2" d="M43.6 20.5H42V20H24v8h11.3c-.8 2.2-2.2 4.2-4.1 5.6l6.2 5.2C37 39.2 44 34 44 24c0-1.3-.1-2.4-.4-3.5z" />
    </svg>
  );
}

export function AccountView() {
  const router = useRouter();
  const { me, user, ready, signOut } = useAuth();
  const nextParam = useQueryParam("next");
  const next = safeNext(nextParam);
  const error = useQueryParam("error");
  const [changingRole, setChangingRole] = useState(false);

  const afterRoleChosen = (role: Role) => {
    setChangingRole(false);
    if (isProfessional(role)) router.push("/apply/");
    else router.push(next);
  };

  if (!ready) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 aria-hidden className="size-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!user) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-xl">Sign in to SwasthyaSetu</CardTitle>
          <CardDescription>
            Patients: signing in is optional. It keeps your chats so you can come back to them.
            You can always{" "}
            <Link href="/patient/" className="text-primary-text underline-offset-4 hover:underline">
              get help without signing in
            </Link>
            .
            <br />
            Doctors, pharmacists and MBBS students: sign in to apply for verification.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          {error && (
            <p role="alert" className="rounded-lg border border-destructive/40 bg-destructive/5 px-3 py-2 text-sm">
              {error}
            </p>
          )}
          {me?.google_enabled ? (
            <Button asChild size="lg" variant="outline" className="w-full">
              <a href={api.auth.googleLoginUrl(next)}>
                <GoogleIcon />
                Continue with Google
              </a>
            </Button>
          ) : (
            !me?.dev_login && (
              <p className="text-sm text-muted-foreground">
                Sign-in isn&apos;t set up on this server yet.
              </p>
            )
          )}
          {me?.dev_login && (
            <DevLoginForm
              onSignedIn={() => {
                // Same as Google sign-in: continue where they were going,
                // unless they still have to choose a role.
                const signedIn = useAuthStore.getState().me?.user;
                if (nextParam && (signedIn?.role || signedIn?.is_admin)) router.push(next);
              }}
            />
          )}
        </CardContent>
      </Card>
    );
  }

  // The admin reviews applications and needs no role of their own.
  if ((user.role === null && !user.is_admin) || changingRole) {
    return (
      <div className="flex flex-col gap-4">
        <div>
          <h1 className="text-xl font-semibold">Welcome, {user.name}</h1>
          <p className="text-sm text-muted-foreground">Who are you? · तपाईं को हुनुहुन्छ?</p>
        </div>
        <RolePicker onChosen={afterRoleChosen} />
        {changingRole && (
          <Button variant="ghost" onClick={() => setChangingRole(false)}>
            Cancel
          </Button>
        )}
      </div>
    );
  }

  const verification = user.verification;
  const verified =
    verification?.status === "approved" && verification.role === user.role;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-xl">
          {user.name}
          {verified && <BadgeCheck aria-label="Verified" className="size-5 text-triage-green" />}
        </CardTitle>
        <CardDescription>
          {user.email} · {user.role ? ROLE_LABEL[user.role] : "Admin"}
          {isProfessional(user.role) &&
            (verified
              ? " · Verified"
              : verification?.status === "pending"
                ? " · Verification pending"
                : verification?.status === "rejected"
                  ? " · Verification needs changes"
                  : " · Not verified yet")}
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        {user.role === "patient" && (
          <Button asChild>
            <Link href="/patient/">Go to chat and your saved chats</Link>
          </Button>
        )}
        {isProfessional(user.role) && (
          <Button asChild>
            <Link href={verified ? "/doctor/" : "/apply/"}>
              <Stethoscope aria-hidden />
              {verified ? "Professional dashboard" : "Verification"}
            </Link>
          </Button>
        )}
        {user.is_admin && (
          <Button asChild variant="secondary">
            <Link href="/admin/">
              <ShieldCheck aria-hidden />
              Admin: review applications
            </Link>
          </Button>
        )}
        {!verified && (
          <Button variant="outline" onClick={() => setChangingRole(true)}>
            Change role
          </Button>
        )}
        <Button
          variant="ghost"
          onClick={async () => {
            await signOut();
            router.push("/");
          }}
        >
          <LogOut aria-hidden />
          Sign out
        </Button>
      </CardContent>
    </Card>
  );
}
