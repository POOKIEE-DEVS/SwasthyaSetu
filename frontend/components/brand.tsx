import { cn } from "@/lib/utils";

/** The ribbon mark from the logo (a transparent PNG cut from the artwork).
 * On a dark background it sits on a light disc, or its navy ribbon would
 * disappear into the page. */
export function BrandMark({ className }: { className?: string }) {
  return (
    // A tiny static asset; next/image adds nothing in a static export.
    // eslint-disable-next-line @next/next/no-img-element
    <img
      src="/brand/mark-128.png"
      alt=""
      width={128}
      height={128}
      className={cn(
        "size-8 shrink-0 dark:rounded-full dark:bg-[oklch(0.96_0.01_255)] dark:p-[3px]",
        className,
      )}
    />
  );
}

/** Mark + wordmark, coloured like the logo: "Swasthya" navy, "Setu" red. */
export function BrandLockup({ className }: { className?: string }) {
  return (
    <span className={cn("flex items-center gap-2", className)} translate="no">
      <BrandMark />
      <span className="text-[17px] font-extrabold tracking-tight">
        <span className="text-foreground">Swasthya</span>
        <span className="text-brand-red">Setu</span>
      </span>
    </span>
  );
}
