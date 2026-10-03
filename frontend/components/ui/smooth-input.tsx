"use client";

/*
 * A text input whose caret glides to its new position with spring physics
 * instead of jumping. Adapted from skiper-ui's "skiper106" (by gxuri):
 * https://skiper-ui.com
 *
 * Changes from the original, for this app:
 * - No dialkit tuning panel (a developer tool that would show to patients).
 * - Safe for the static export: nothing touches `navigator` or `document`
 *   while the page is pre-rendered.
 * - Text only (no password mode), brand colours, and the caret width and
 *   height follow the input's own font.
 * - Respects reduced motion: the caret moves instantly.
 */

import {
  type ComponentPropsWithoutRef,
  type Ref,
  useEffect,
  useImperativeHandle,
  useLayoutEffect,
  useRef,
} from "react";
import { motion, useMotionValue, useReducedMotion, useSpring } from "motion/react";

import { cn } from "@/lib/utils";

type SmoothInputProps = Omit<ComponentPropsWithoutRef<"input">, "type"> & {
  ref?: Ref<HTMLInputElement>;
  wrapperClassName?: string;
};

const SPRING = { stiffness: 500, damping: 30, mass: 0.5 };
const INSTANT = { stiffness: 10000, damping: 100, mass: 0.1 };

export function SmoothInput({
  ref,
  className,
  wrapperClassName,
  value,
  onChange,
  onBlur,
  onFocus,
  ...props
}: SmoothInputProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const measureRef = useRef<HTMLSpanElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  useImperativeHandle(ref, () => inputRef.current as HTMLInputElement);

  const reduceMotion = useReducedMotion();
  const caretX = useMotionValue(0);
  const caretOpacity = useMotionValue(0);
  const springX = useSpring(caretX, reduceMotion ? INSTANT : SPRING);

  /** Width of the text before the caret, measured with the input's font. */
  const measure = (input: HTMLInputElement, text: string): number => {
    const span = measureRef.current;
    const styles = window.getComputedStyle(input);
    const paddingLeft = parseFloat(styles.paddingLeft) || 0;
    if (!span) return paddingLeft;
    span.style.font = `${styles.fontStyle} ${styles.fontWeight} ${styles.fontSize} ${styles.fontFamily}`;
    span.style.letterSpacing = styles.letterSpacing;
    span.style.fontFeatureSettings = styles.fontFeatureSettings;
    span.textContent = text;
    return text.length > 0 ? span.offsetWidth + paddingLeft : paddingLeft - 1;
  };

  const update = (input: HTMLInputElement) => {
    const start = input.selectionStart ?? 0;
    const end = input.selectionEnd ?? 0;
    const index =
      start === end ? start : input.selectionDirection === "backward" ? start : end;
    const absolute = measure(input, input.value.slice(0, index));

    // Keep the caret in view when the text is wider than the box.
    const styles = window.getComputedStyle(input);
    const paddingLeft = parseFloat(styles.paddingLeft) || 0;
    const paddingRight = parseFloat(styles.paddingRight) || 0;
    const visibleRight = input.scrollLeft + input.clientWidth - paddingRight;
    if (absolute > visibleRight) {
      input.scrollLeft = absolute - input.clientWidth + paddingRight;
    } else if (absolute < input.scrollLeft + paddingLeft) {
      input.scrollLeft = Math.max(0, absolute - paddingLeft);
    }

    const position = absolute - input.scrollLeft;
    const maxX = input.clientWidth - paddingRight;
    caretX.set(Math.min(position, maxX));
    // Hide the custom caret while text is selected (the selection shows).
    caretOpacity.set(start !== end ? 0 : 1);
  };

  // Event listeners call the latest `update` through this ref; it is
  // refreshed after every render, never during one.
  const updateRef = useRef(update);
  useLayoutEffect(() => {
    updateRef.current = update;
  });

  // Re-place the caret when the value changes from outside (e.g. cleared
  // after sending).
  useEffect(() => {
    const input = inputRef.current;
    if (input && document.activeElement === input) updateRef.current(input);
  }, [value]);

  useEffect(() => {
    const input = inputRef.current;
    const container = containerRef.current;
    if (!input || !container) return;

    const ifFocused = () => {
      if (document.activeElement === input) updateRef.current(input);
    };
    const onSelection = () => {
      if (document.activeElement === input) requestAnimationFrame(ifFocused);
    };

    document.addEventListener("selectionchange", onSelection);
    input.addEventListener("scroll", ifFocused);
    void document.fonts?.ready.then(ifFocused);
    const observer = new ResizeObserver(ifFocused);
    observer.observe(container);
    return () => {
      document.removeEventListener("selectionchange", onSelection);
      input.removeEventListener("scroll", ifFocused);
      observer.disconnect();
    };
  }, []);

  return (
    <div className={cn("relative min-w-0", wrapperClassName)}>
      <div
        ref={containerRef}
        className="relative grid grid-cols-1"
        // The native caret is hidden; the animated one replaces it.
        style={{ caretColor: "transparent" }}
      >
        <input
          {...props}
          ref={inputRef}
          type="text"
          value={value}
          className={cn(
            "col-start-1 row-start-1 w-full min-w-0 bg-transparent outline-none placeholder:text-muted-foreground",
            className,
          )}
          onChange={(event) => {
            onChange?.(event);
            const target = event.target;
            requestAnimationFrame(() => updateRef.current(target));
          }}
          onFocus={(event) => {
            updateRef.current(event.target);
            onFocus?.(event);
          }}
          onBlur={(event) => {
            caretOpacity.set(0);
            onBlur?.(event);
          }}
        />
        <span
          ref={measureRef}
          aria-hidden
          className="pointer-events-none invisible absolute left-0 top-0 whitespace-pre"
        />
        <motion.span
          aria-hidden
          className="pointer-events-none col-start-1 row-start-1 h-[1.15em] w-[2px] self-center rounded-full bg-primary"
          style={{ x: springX, opacity: caretOpacity }}
        />
      </div>
    </div>
  );
}
