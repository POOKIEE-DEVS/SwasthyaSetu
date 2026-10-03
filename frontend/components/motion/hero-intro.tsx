"use client";

import { useLayoutEffect } from "react";

/** How long after the first paint the hero entrance may still start (the
 * CSS fallback shows the hero at 600ms, so never replay it after that). */
const LATE_MS = 550;
/** The entrance settles in about 1.4s; this leaves room for the split. */
const DONE_MS = 2000;

function msSinceFirstPaint(): number {
  const paint = performance.getEntriesByName?.("first-contentful-paint")[0];
  return performance.now() - (paint?.startTime ?? 0);
}

/**
 * Marks the hero entrance as done (.hero-intro-done on <html>), which
 * switches off every hero entrance rule in globals.css. It plays once per
 * page load: coming back to the landing page shows it already settled.
 * If the script arrives late, the entrance is skipped instead of replaying
 * over content the CSS fallback has already shown.
 */
export function HeroIntro() {
  useLayoutEffect(() => {
    const root = document.documentElement;
    if (root.classList.contains("hero-intro-done")) return;
    if (msSinceFirstPaint() > LATE_MS) {
      root.classList.add("hero-intro-done");
      return;
    }
    const timer = window.setTimeout(() => root.classList.add("hero-intro-done"), DONE_MS);
    return () => window.clearTimeout(timer);
  }, []);

  return null;
}
