"use client";

import { useEffect } from "react";
import { create } from "zustand";

import { api, type Me, type ProfessionalRole, type User } from "@/lib/api";
import { useChatStore } from "@/lib/store/chat";

/**
 * Who is signed in, loaded once from /api/v1/auth/me and shared by every
 * page. Signing in is optional for patients, so "signed out" is a normal,
 * fully working state, not an error.
 */
type AuthState = {
  status: "idle" | "loading" | "ready" | "error";
  me: Me | null;
  load: () => Promise<void>;
  signOut: () => Promise<void>;
};

export const useAuthStore = create<AuthState>()((set, get) => ({
  status: "idle",
  me: null,

  load: async () => {
    set({ status: "loading" });
    try {
      set({ me: await api.auth.me(), status: "ready" });
    } catch {
      // Offline or server down: behave as signed out. Chat still works.
      set({ status: "error" });
    }
  },

  signOut: async () => {
    await api.auth.logout().catch(() => {});
    // A shared phone must not show the last person's private chat.
    useChatStore.getState().reset();
    const me = get().me;
    set({ me: me ? { ...me, user: null } : null });
  },
}));

/** The auth state, loading it on first use. */
export function useAuth() {
  const state = useAuthStore();
  useEffect(() => {
    if (useAuthStore.getState().status === "idle") void useAuthStore.getState().load();
  }, []);
  const user = state.me?.user ?? null;
  return {
    ...state,
    user,
    ready: state.status === "ready" || state.status === "error",
    verifiedRole: verifiedRole(user),
  };
}

const PROFESSIONAL: ProfessionalRole[] = ["doctor", "pharmacist", "nurse", "paramedic", "student"];

export function isProfessional(role: User["role"]): role is ProfessionalRole {
  return role !== null && (PROFESSIONAL as string[]).includes(role);
}

/** The role this user is verified for, or null. Mirrors the server's rule. */
export function verifiedRole(user: User | null): ProfessionalRole | null {
  const v = user?.verification;
  return v && v.status === "approved" && user?.role === v.role ? v.role : null;
}
