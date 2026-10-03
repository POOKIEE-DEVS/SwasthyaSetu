import { cn } from "@/lib/utils";

const MARK = "/brand/mark-128.png";

/**
 * The ribbon mark from the logo.
 *
 * "mono" (the default) draws the mark's exact silhouette in one colour
 * through a CSS mask, forest on the page, so it sits naturally in the warm,
 * mostly neutral palette and works in dark mode. The logo files are never
 * altered: "color" shows the original navy and red artwork, which is also
 * what the favicon and app icons use.
 */
export function BrandMark({
  className,
  tone = "mono",
}: {
  className?: string;
  tone?: "mono" | "color";
}) {
  if (tone === "color") {
    return (
      // A tiny static asset; next/image adds nothing in a static export.
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={MARK}
        alt=""
        width={128}
        height={128}
        className={cn("size-8 shrink-0", className)}
      />
    );
  }
  return (
    <span
      aria-hidden
      className={cn(
        "inline-block size-8 shrink-0 bg-primary [mask:url(/brand/mark-128.png)_center/contain_no-repeat]",
        className,
      )}
    />
  );
}

/** Mark + wordmark: one weight, one colour, so the mark carries the brand. */
export function BrandLockup({ className }: { className?: string }) {
  return (
    <span className={cn("flex items-center gap-2", className)} translate="no">
      <BrandMark className="size-7" />
      <span className="text-[17px] font-semibold tracking-[-0.01em] text-foreground">
        SwasthyaSetu
      </span>
    </span>
  );
}
