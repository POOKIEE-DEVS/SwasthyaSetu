"use client";

import { create } from "zustand";

/**
 * Connectivity state, shared across the app.
 *
 * This is the seam Offline Emergency Mode (Week 18) is built on. Connectivity
 * is not incidental to this product: the users it targets lose signal
 * routinely, and the app has to keep showing cached first-aid articles,
 * emergency contacts, and GPS rather than an error page.
 *
 * `queuedReports` counts symptom reports logged while offline and not yet
 * synced. It lives in the store rather than in a component so the count
 * survives navigation, and so the sync-on-reconnect handler has one place to
 * read it from.
 */
type ConnectionState = {
  /** Null until the client has mounted -- the server cannot know this. */
  isOnline: boolean | null;
  queuedReports: number;
  setOnline: (online: boolean) => void;
  enqueueReport: () => void;
  clearQueue: () => void;
};

export const useConnectionStore = create<ConnectionState>((set) => ({
  isOnline: null,
  queuedReports: 0,
  setOnline: (online) => set({ isOnline: online }),
  enqueueReport: () =>
    set((state) => ({ queuedReports: state.queuedReports + 1 })),
  clearQueue: () => set({ queuedReports: 0 }),
}));
