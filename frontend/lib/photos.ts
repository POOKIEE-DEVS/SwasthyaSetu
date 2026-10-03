/**
 * Photographs for the landing page.
 *
 * The page is designed around authentic, documentary photography from
 * Nepal: real homes and health posts, natural daylight, quiet moments of
 * care. None has been licensed yet, so every slot is empty. To add one,
 * put the file in public/photos/ and fill in `photo`, with an alt text
 * that describes what the actual picture shows.
 *
 * While a slot is empty, development builds show a placeholder carrying
 * the brief, and production builds leave the slot out, so the live page
 * never shows an empty frame. No stock hospital photos, no posed medical
 * teams, no AI-generated people.
 */
export type Photo = { src: string; alt: string; width: number; height: number };

export type PhotoSlot = "careAtHome";

/** `brief` says what the photo should show; `photo` is null until one is licensed. */
export const PHOTO_SLOTS: Record<PhotoSlot, { brief: string; photo: Photo | null }> = {
  careAtHome: {
    brief:
      "A parent and child at home in the evening, outside the Kathmandu Valley, a phone in hand. Natural window light, documentary, not posed.",
    photo: null,
  },
};
