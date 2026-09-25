"use client";

import { useMemo, useSyncExternalStore } from "react";

import type { CallTicket } from "@/lib/api";

/**
 * The active call ticket, kept in this tab's sessionStorage so a refresh
 * rejoins the same call instead of stranding it. The token only grants
 * access to that one consultation.
 *
 * Exposed as an external store (useSyncExternalStore). The server snapshot
 * is `null`, so the static export hydrates cleanly and the stored ticket
 * appears on the client without a setState-in-effect round trip.
 */

type Role = CallTicket["role"];

const key = (role: Role) => `swasthyasetu-call-${role}`;
// Fallback for when sessionStorage throws (some private-browsing modes), so
// a call can still start; it just won't survive a refresh.
const memory = new Map<string, string>();
const listeners = new Set<() => void>();

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function readRaw(role: Role): string | null {
  try {
    return sessionStorage.getItem(key(role)) ?? memory.get(key(role)) ?? null;
  } catch {
    return memory.get(key(role)) ?? null;
  }
}

export function saveTicket(ticket: CallTicket): void {
  const raw = JSON.stringify(ticket);
  try {
    sessionStorage.setItem(key(ticket.role), raw);
  } catch {
    memory.set(key(ticket.role), raw);
  }
  listeners.forEach((listener) => listener());
}

export function clearTicket(role: Role): void {
  try {
    sessionStorage.removeItem(key(role));
  } catch {
    // ignore
  }
  memory.delete(key(role));
  listeners.forEach((listener) => listener());
}

/** The current ticket for this role, or null. */
export function useTicket(role: Role): CallTicket | null {
  // The raw string is the snapshot: it compares stably by value, unlike a
  // freshly parsed object.
  const raw = useSyncExternalStore(
    subscribe,
    () => readRaw(role),
    () => null,
  );
  return useMemo(() => {
    if (!raw) return null;
    try {
      return JSON.parse(raw) as CallTicket;
    } catch {
      return null;
    }
  }, [raw]);
}
