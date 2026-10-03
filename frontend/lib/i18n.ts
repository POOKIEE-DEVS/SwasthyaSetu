"use client";

import { useEffect, useSyncExternalStore } from "react";

import { COPY, type Copy } from "@/lib/copy";

export type Lang = "en" | "ne";

const KEY = "swasthyasetu-lang";
const listeners = new Set<() => void>();
// Fallback for when localStorage throws (some private-browsing modes).
let memory: Lang | null = null;

function read(): Lang {
  if (memory) return memory;
  try {
    return localStorage.getItem(KEY) === "ne" ? "ne" : "en";
  } catch {
    return "en";
  }
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function setLang(lang: Lang): void {
  memory = lang;
  try {
    localStorage.setItem(KEY, lang);
  } catch {
    // Kept in memory for this visit.
  }
  listeners.forEach((listener) => listener());
}

/**
 * The visitor's language. The static export renders English (the server
 * snapshot), then the saved choice appears on the client without a
 * hydration mismatch. Also keeps <html lang> in step for screen readers.
 */
export function useLang(): Lang {
  const lang = useSyncExternalStore(subscribe, read, () => "en" as Lang);
  useEffect(() => {
    document.documentElement.lang = lang;
  }, [lang]);
  return lang;
}

export function useCopy(): { lang: Lang; t: Copy } {
  const lang = useLang();
  return { lang, t: COPY[lang] };
}
