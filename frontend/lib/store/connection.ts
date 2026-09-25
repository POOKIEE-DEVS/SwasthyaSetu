"use client";

import { create } from "zustand";

/** Browser connectivity. `null` until the client has mounted. */
type ConnectionState = {
  isOnline: boolean | null;
  setOnline: (online: boolean) => void;
};

export const useConnectionStore = create<ConnectionState>((set) => ({
  isOnline: null,
  setOnline: (online) => set({ isOnline: online }),
}));
