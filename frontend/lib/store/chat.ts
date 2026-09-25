"use client";

import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

import { api, ApiError, type ChatMessage } from "@/lib/api";

/**
 * The patient's conversation with the AI.
 *
 * Kept in this tab's sessionStorage, so a page refresh mid-demo does not
 * lose it. The backend is stateless for chat: the whole conversation is sent
 * with each message, and the server trims it to recent turns.
 *
 * `skipHydration` avoids a server/client mismatch in the static export: the
 * patient page calls `useChatStore.persist.rehydrate()` after mount.
 */
type ChatState = {
  messages: ChatMessage[];
  pending: boolean;
  error: string | null;
  /** The latest user message mentioned an obvious emergency. */
  urgent: boolean;
  send: (text: string) => Promise<void>;
  retry: () => Promise<void>;
  reset: () => void;
};

async function ask(
  messages: ChatMessage[],
  set: (partial: Partial<ChatState>) => void,
): Promise<void> {
  set({ pending: true, error: null });
  try {
    const { reply, urgent } = await api.chat(messages);
    set({
      messages: [...messages, { role: "assistant", content: reply }],
      urgent,
      pending: false,
    });
  } catch (error) {
    set({
      pending: false,
      error:
        error instanceof ApiError ? error.message : "Something went wrong. Please try again.",
    });
  }
}

export const useChatStore = create<ChatState>()(
  persist(
    (set, get) => ({
      messages: [],
      pending: false,
      error: null,
      urgent: false,

      send: async (text) => {
        const content = text.trim();
        if (!content || get().pending) return;
        const messages: ChatMessage[] = [...get().messages, { role: "user", content }];
        set({ messages });
        await ask(messages, set);
      },

      // Re-sends the conversation as it stands. The unanswered user message
      // is already the last entry.
      retry: async () => {
        const { messages, pending } = get();
        if (pending || messages.at(-1)?.role !== "user") return;
        await ask(messages, set);
      },

      reset: () => set({ messages: [], pending: false, error: null, urgent: false }),
    }),
    {
      name: "swasthyasetu-chat",
      storage: createJSONStorage(() => sessionStorage),
      partialize: (state) => ({ messages: state.messages, urgent: state.urgent }),
      skipHydration: true,
    },
  ),
);

/** The conversation as plain text, for sharing with the doctor. */
export function transcript(messages: ChatMessage[], maxChars = 5000): string | null {
  if (messages.length === 0) return null;
  const text = messages
    .map((m) => `${m.role === "user" ? "Patient" : "AI assistant"}: ${m.content}`)
    .join("\n\n");
  // Keep the most recent part if it is too long.
  return text.length > maxChars ? `…${text.slice(-maxChars)}` : text;
}
