"use client";

import { useEffect, useState } from "react";
import { ChevronLeft, ChevronRight, Pause, Play } from "lucide-react";
import { useReducedMotion } from "motion/react";

import { Button } from "@/components/ui/button";
import { TESTIMONIALS } from "@/lib/testimonials";

const ROTATE_MS = 7000;

/**
 * An accessible testimonials carousel: previous / next and pause controls,
 * no rotation under reduced motion or while it has focus or the pointer,
 * and the slide position announced to screen readers. Renders nothing until
 * lib/testimonials.ts has real entries.
 */
export function Testimonials() {
  const [index, setIndex] = useState(0);
  const [paused, setPaused] = useState(false);
  const [hovered, setHovered] = useState(false);
  const reduceMotion = useReducedMotion();
  const count = TESTIMONIALS.length;
  const rotating = count > 1 && !paused && !hovered && !reduceMotion;

  useEffect(() => {
    if (!rotating) return;
    const id = setInterval(() => setIndex((i) => (i + 1) % count), ROTATE_MS);
    return () => clearInterval(id);
  }, [rotating, count]);

  if (count === 0) return null;
  const t = TESTIMONIALS[index];

  return (
    <section aria-labelledby="voices-title" className="mx-auto max-w-4xl px-4 py-20">
      <h2 id="voices-title" className="text-3xl font-bold tracking-tight sm:text-4xl">
        What people say
      </h2>
      <div
        className="mt-8 rounded-[20px] bg-card p-8 shadow-raised sm:p-10"
        aria-roledescription="carousel"
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
        onFocus={() => setHovered(true)}
        onBlur={() => setHovered(false)}
      >
        <figure aria-live={rotating ? "off" : "polite"} aria-roledescription="slide">
          <blockquote className="line-clamp-3 text-xl italic leading-relaxed text-muted-foreground sm:text-2xl">
            &ldquo;{t.quote}&rdquo;
          </blockquote>
          <figcaption className="mt-6 flex items-center gap-3">
            {t.photo && (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={t.photo}
                alt=""
                width={48}
                height={48}
                loading="lazy"
                className="size-12 rounded-full object-cover"
              />
            )}
            <span>
              <span className="block font-bold">{t.name}</span>
              <span className="block text-sm text-muted-foreground">{t.role}</span>
            </span>
          </figcaption>
        </figure>
        {count > 1 && (
          <div className="mt-8 flex items-center gap-2">
            <Button
              variant="outline"
              size="icon"
              aria-label="Previous testimonial"
              onClick={() => setIndex((i) => (i - 1 + count) % count)}
            >
              <ChevronLeft aria-hidden />
            </Button>
            <Button
              variant="outline"
              size="icon"
              aria-label="Next testimonial"
              onClick={() => setIndex((i) => (i + 1) % count)}
            >
              <ChevronRight aria-hidden />
            </Button>
            {!reduceMotion && (
              <Button
                variant="ghost"
                size="icon"
                aria-label={paused ? "Resume rotation" : "Pause rotation"}
                onClick={() => setPaused((p) => !p)}
              >
                {paused ? <Play aria-hidden /> : <Pause aria-hidden />}
              </Button>
            )}
            <span className="tabular ml-auto text-sm text-muted-foreground" aria-live="polite">
              {index + 1} of {count}
            </span>
          </div>
        )}
      </div>
    </section>
  );
}
