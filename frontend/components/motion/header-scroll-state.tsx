"use client";

import { useEffect } from "react";

/**
 * Sets `data-scrolled` on the landing page header once the page has moved,
 * which brings in its hairline (see .site-header-home in globals.css).
 * One passive listener, at most one DOM write per frame, and only when the
 * state actually changes.
 */
export function HeaderScrollState() {
  useEffect(() => {
    const header = document.querySelector<HTMLElement>(".site-header-home");
    if (!header) return;
    let frame = 0;
    const update = () => {
      frame = 0;
      header.toggleAttribute("data-scrolled", window.scrollY > 8);
    };
    const onScroll = () => {
      if (!frame) frame = requestAnimationFrame(update);
    };
    update();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => {
      window.removeEventListener("scroll", onScroll);
      cancelAnimationFrame(frame);
    };
  }, []);

  return null;
}
