import { PHOTO_SLOTS, type PhotoSlot } from "@/lib/photos";
import { cn } from "@/lib/utils";

/**
 * A full-bleed documentary photograph, or nothing. While the slot is empty
 * (see lib/photos.ts), development builds show the brief for the photo in
 * its place; production builds render nothing, and the section around it
 * is composed to stand on its own.
 *
 * A photo drifts very slightly against the page as it scrolls (desktop
 * only, StringParallax at 0.12): enough to feel smooth, too little to
 * notice as an effect. Three layers so nothing fights over a transform:
 * the frame reveals, the middle layer belongs to StringTune, and the image
 * is oversized so the drift never shows an edge.
 */
export function EditorialPhoto({ slot, className }: { slot: PhotoSlot; className?: string }) {
  const { brief, photo } = PHOTO_SLOTS[slot];

  if (photo) {
    return (
      <div data-reveal className={cn("relative w-full overflow-hidden rounded-[4px]", className)}>
        {/* "parallax[]": active in the native scroll mode this site uses.
            StringTune never starts on touch devices, so phones stay still. */}
        <div
          className="absolute inset-x-0 -inset-y-[8%]"
          data-string="parallax[]"
          data-string-parallax="0.12"
        >
          {/* A static export serves images as they are; next/image adds nothing. */}
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={photo.src}
            alt={photo.alt}
            width={photo.width}
            height={photo.height}
            loading="lazy"
            className="size-full object-cover"
          />
        </div>
      </div>
    );
  }

  if (process.env.NODE_ENV === "production") return null;

  return (
    <div
      className={cn(
        "flex w-full items-end rounded-[4px] bg-[#e7e3d8] p-5 dark:bg-white/[0.04]",
        className,
      )}
    >
      <p className="max-w-md text-xs leading-relaxed text-muted-foreground">
        <span className="font-semibold">Photograph to come</span> (shown in development only).{" "}
        {brief}
      </p>
    </div>
  );
}
