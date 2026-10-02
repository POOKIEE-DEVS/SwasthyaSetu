"use client";

import Link from "next/link";
import { LogIn } from "lucide-react";

import { useAuth } from "@/lib/store/auth";

function initials(name: string): string {
  const parts = name.trim().split(/\s+/);
  return ((parts[0]?.[0] ?? "") + (parts.length > 1 ? (parts.at(-1)?.[0] ?? "") : "")).toUpperCase();
}

/** Header entry point: "Sign in", or the signed-in user's initials. */
export function AccountButton() {
  const { user, ready } = useAuth();

  if (!ready) return <span className="size-9" aria-hidden />;

  if (!user) {
    return (
      <Link
        href="/account/"
        className="flex h-9 items-center gap-1.5 rounded-md border px-3 text-sm font-medium hover:bg-accent"
      >
        <LogIn aria-hidden className="size-4" />
        Sign in
      </Link>
    );
  }

  return (
    <Link
      href="/account/"
      aria-label={`Account: ${user.name}`}
      title={user.email}
      className="flex size-9 items-center justify-center rounded-full bg-primary text-sm font-semibold text-primary-foreground"
    >
      {initials(user.name) || "?"}
    </Link>
  );
}
