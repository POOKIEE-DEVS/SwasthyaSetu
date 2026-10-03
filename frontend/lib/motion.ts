/**
 * Where the landing page's scroll-linked motion runs.
 *
 * StringTune (the hero line split, the care-journey progress) only starts on
 * a desktop-class device: a fine pointer, at least tablet width, and no
 * request for reduced motion. Phones and tablets get the same page with
 * native scrolling, short CSS reveals and no per-frame scroll work, which is
 * what matters there: speed, readability and battery during a call.
 *
 * globals.css repeats this media query for the CSS that depends on it;
 * keep the two in step.
 */
export const MOTION_RUNTIME_QUERY =
  "(min-width: 768px) and (hover: hover) and (pointer: fine) and (prefers-reduced-motion: no-preference)";

export const REDUCED_MOTION_QUERY = "(prefers-reduced-motion: reduce)";

export function matches(query: string): boolean {
  return typeof window !== "undefined" && (window.matchMedia?.(query).matches ?? false);
}
