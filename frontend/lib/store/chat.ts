"use client";

import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

import { api, ApiError, type ChatDetail, type ChatMessage } from "@/lib/api";

/**
 * The patient's conversation with the AI.
 *
 * Kept in this tab's sessionStorage, so a page refresh mid-demo does not
 * lose it. The whole conversation is sent with each message, and the
 * server trims it to recent turns.
 *
 * Signed-in patients also get it saved on the server: `chatId` is the
 * saved chat this conversation continues. Guests have no chatId and
 * nothing is stored.
 *
 * `skipHydration` avoids a server/client mismatch in the static export: the
 * patient page calls `useChatStore.persist.rehydrate()` after mount.
 */
type ChatState = {
  messages: ChatMessage[];
  chatId: string | null;
  pending: boolean;
  error: string | null;
  /** The latest user message mentioned an obvious emergency. */
  urgent: boolean;
  send: (text: string) => Promise<void>;
  retry: () => Promise<void>;
  reset: () => void;
  /** Open a saved chat from the history list. */
  open: (chat: ChatDetail) => void;
  /** After signing in: save the conversation started as a guest. */
  saveGuestChat: () => Promise<void>;
};

async function ask(
  messages: ChatMessage[],
  chatId: string | null,
  set: (partial: Partial<ChatState>) => void,
): Promise<void> {
  set({ pending: true, error: null });
  try {
    const { reply, urgent, chat_id } = await api.chat(messages, chatId);
    set({
      messages: [...messages, { role: "assistant", content: reply }],
      // Null for guests; the saved chat's id for signed-in patients.
      chatId: chat_id ?? chatId,
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
      chatId: null,
      pending: false,
      error: null,
      urgent: false,

      send: async (text) => {
        const content = text.trim();
        if (!content || get().pending) return;
        const messages: ChatMessage[] = [...get().messages, { role: "user", content }];
        set({ messages });
        await ask(messages, get().chatId, set);
      },

      // Re-sends the conversation as it stands. The unanswered user message
      // is already the last entry.
      retry: async () => {
        const { messages, pending, chatId } = get();
        if (pending || messages.at(-1)?.role !== "user") return;
        await ask(messages, chatId, set);
      },

      reset: () =>
        set({ messages: [], chatId: null, pending: false, error: null, urgent: false }),

      open: (chat) =>
        set({
          messages: chat.messages,
          chatId: chat.id,
          pending: false,
          error: null,
          urgent: false,
        }),

      saveGuestChat: async () => {
        const { messages, chatId, pending } = get();
        // Only a finished exchange: an unanswered question is saved by its
        // retry instead.
        if (chatId || pending || messages.at(-1)?.role !== "assistant") return;
        try {
          const saved = await api.chats.save(messages);
          if (get().messages === messages) set({ chatId: saved.id });
        } catch {
          // Not saved; the conversation itself is unaffected.
        }
      },
    }),
    {
      name: "swasthyasetu-chat",
      storage: createJSONStorage(() => sessionStorage),
      partialize: (state) => ({
        messages: state.messages,
        chatId: state.chatId,
        urgent: state.urgent,
      }),
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
