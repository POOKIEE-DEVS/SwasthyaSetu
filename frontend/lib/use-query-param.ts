"use client";

import { useSyncExternalStore } from "react";

const noSubscribe = () => () => {};

/**
 * One query-string value, read on the client only. The static export
 * renders pages at build time, where there is no URL; the server snapshot
 * is null, so hydration matches and the value appears right after.
 */
export function useQueryParam(name: string): string | null {
  return useSyncExternalStore(
    noSubscribe,
    () => new URLSearchParams(window.location.search).get(name),
    () => null,
  );
}
