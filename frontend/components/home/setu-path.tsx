import { cn } from "@/lib/utils";

/**
 * Give it an explicit width and height: an SVG doesn't stretch between
 * left/right (or top/bottom) insets the way a normal box does. No
 * `vector-effect: non-scaling-stroke`: it makes the dash pattern ignore
 * pathLength, which breaks the draw-in.
 *
 * The Setu line: one thin, gently curving path that connects the steps of
 * the journey. It draws itself in as it scrolls into view (see .setu-line
 * in globals.css) and is simply drawn when motion is reduced or the browser
 * lacks scroll-driven animations. Decorative, so hidden from screen readers.
 */
export function SetuPath({
  orientation,
  className,
}: {
  orientation: "horizontal" | "vertical";
  className?: string;
}) {
  const horizontal = orientation === "horizontal";
  return (
    <svg
      aria-hidden
      focusable="false"
      className={cn("pointer-events-none overflow-visible", className)}
      viewBox={horizontal ? "0 0 1000 24" : "0 0 24 1000"}
      preserveAspectRatio="none"
    >
      <path
        className="setu-line"
        pathLength={1}
        d={
          horizontal
            ? "M0 12 C 160 2, 330 22, 500 12 S 840 2, 1000 12"
            : "M12 0 C 2 160, 22 330, 12 500 S 2 840, 12 1000"
        }
        fill="none"
        stroke="var(--fresh)"
        strokeWidth={2}
        strokeLinecap="round"
      />
    </svg>
  );
}
