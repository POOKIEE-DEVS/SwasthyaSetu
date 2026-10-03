import { cn } from "@/lib/utils";

/**
 * The Setu: one hairline suspension bridge between home and care, like
 * the footbridges (jholunge pul) that join villages across Nepal's rivers.
 * The only graphic on the landing page.
 *
 * Decorative: the headline beside it says the same thing, so it is hidden
 * from screen readers. The curve stretches to any width while its stroke
 * stays 1.5px (non-scaling-stroke); the two ends are HTML so they stay
 * round. It draws in once from left to right (.setu-bridge in globals.css).
 */
export function SetuBridge({ className }: { className?: string }) {
  return (
    <div aria-hidden className={cn("select-none", className)}>
      {/* Above the ends: the curve falls away below them. */}
      <div className="mb-3 flex justify-between text-sm text-muted-foreground">
        <span>
          Home · <span lang="ne">घर</span>
        </span>
        <span>
          Care · <span lang="ne">उपचार</span>
        </span>
      </div>
      <div className="setu-bridge relative h-14 sm:h-20">
        <svg
          focusable="false"
          className="absolute inset-0 size-full overflow-visible"
          viewBox="0 0 1000 100"
          preserveAspectRatio="none"
        >
          <path
            d="M0 6 Q 500 128 1000 6"
            fill="none"
            stroke="var(--fresh)"
            strokeWidth={1.5}
            vectorEffect="non-scaling-stroke"
          />
        </svg>
        <span className="absolute left-0 top-[6%] size-[7px] -translate-y-1/2 rounded-full bg-primary" />
        <span className="absolute right-0 top-[6%] size-[7px] -translate-y-1/2 rounded-full bg-primary" />
      </div>
    </div>
  );
}
