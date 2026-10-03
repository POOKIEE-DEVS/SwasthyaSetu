"use client";

import { useLayoutEffect } from "react";

import { matches, REDUCED_MOTION_QUERY } from "@/lib/motion";

/**
 * One-time reveals for the landing page, on every device.
 *
 * Elements marked `data-reveal` start hidden only once this has run (it adds
 * `reveal-ready` to <html>), so without JavaScript, or with reduced
 * motion, everything is simply visible. Anything already on screen when
 * this runs is marked revealed in the same frame, so it never disappears;
 * anything reached later (including by a deep link that scrolls after
 * loading) reveals as it arrives. Hiding is instant, only the reveal
 * animates. Each element reveals once and stays; scrolling back up does
 * not replay it.
 *
 * CSS owns the animation (see "Reveals" in globals.css). Put `data-reveal`
 * on wrappers, never on elements with their own transform or transition.
 */
export function RevealObserver() {
  useLayoutEffect(() => {
    if (matches(REDUCED_MOTION_QUERY) || !("IntersectionObserver" in window)) return;

    const pending = Array.from(
      document.querySelectorAll<HTMLElement>("[data-reveal]:not([data-revealed])"),
    );
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue;
          entry.target.setAttribute("data-revealed", "");
          observer.unobserve(entry.target);
        }
      },
      // Reveal a little before the element is fully in view.
      { rootMargin: "0px 0px -12% 0px" },
    );

    for (const el of pending) {
      if (el.getBoundingClientRect().top < window.innerHeight) {
        el.setAttribute("data-revealed", "");
      } else {
        observer.observe(el);
      }
    }
    document.documentElement.classList.add("reveal-ready");

    return () => observer.disconnect();
  }, []);

  return null;
}
