/**
 * Testimonials shown on the home page.
 *
 * Only add REAL quotes from real people who agreed to be quoted (a doctor
 * who tried the app, a pilot user), with their actual name and role. The
 * section stays hidden while this list is empty: invented testimonials
 * would be fake social proof.
 *
 * `photo` is optional: a path under /public, e.g. "/people/anita.jpg".
 */
export type Testimonial = {
  quote: string;
  name: string;
  role: string;
  photo?: string;
};

export const TESTIMONIALS: Testimonial[] = [];
