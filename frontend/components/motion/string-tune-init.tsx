"use client";

import { useEffect } from "react";
import StringTune, { StringParallax, StringProgress, StringSplit } from "@fiddle-digital/string-tune";

import { matches, MOTION_RUNTIME_QUERY } from "@/lib/motion";

// StringTune is a singleton with no public stop or destroy, so it is started
// at most once per page load. This survives React's double effects in
// development and remounts after client-side navigation.
let started = false;

/**
 * The one StringTune runtime. Rendered by the landing page only; it finds
 * its elements through `data-string` attributes and picks up new ones on
 * its own (it observes the DOM), so nothing else initialises it.
 *
 * - Native scrolling everywhere: StringTune's smooth mode would take over
 *   wheel, keyboard and anchor scrolling, and a healthcare page should
 *   scroll exactly like the browser does.
 * - Only the modules in use: Split (hero headline lines), Progress (the
 *   care journey) and Parallax (the documentary photo, once there is one).
 * - Not started on touch devices or with reduced motion (MOTION_RUNTIME_QUERY);
 *   the page is complete without it, so a failure here only costs motion.
 */
export function StringTuneInit() {
  useEffect(() => {
    if (started || !matches(MOTION_RUNTIME_QUERY)) return;
    started = true;
    try {
      const tune = StringTune.getInstance();
      tune.scrollDesktopMode = "default";
      tune.scrollMobileMode = "default";
      tune.use(StringSplit);
      tune.use(StringProgress);
      tune.use(StringParallax);
      tune.start(60);
    } catch (error) {
      // The CSS fallbacks show everything without it.
      console.warn("Motion runtime unavailable", error);
    }
  }, []);

  return null;
}
