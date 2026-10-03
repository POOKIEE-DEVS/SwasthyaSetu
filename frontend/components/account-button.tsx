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

  if (!ready) return <span className="size-11 sm:size-9" aria-hidden />;

  if (!user) {
    return (
      <Link
        href="/account/"
        className="flex h-11 items-center gap-1.5 rounded-full border border-input bg-card px-3.5 text-sm font-bold shadow-raised-sm transition-colors duration-200 hover:border-primary/50 hover:bg-accent sm:h-9"
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
      className="flex size-11 items-center justify-center rounded-full bg-primary text-[13px] font-bold text-primary-foreground shadow-raised-sm transition-opacity duration-200 hover:opacity-90 sm:size-9"
    >
      {initials(user.name) || "?"}
    </Link>
  );
}
