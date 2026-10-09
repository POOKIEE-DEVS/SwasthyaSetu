"use client";

import { useEffect, useState } from "react";
import { Loader2, MessageSquare, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { api, ApiError, type ChatSummary } from "@/lib/api";
import { useChatStore } from "@/lib/store/chat";

function when(timestamp: number): string {
  return new Date(timestamp * 1000).toLocaleString(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  });
}

/** A signed-in patient's saved chats. Picking one opens it in the chat. */
export function ChatHistory({ onOpened }: { onOpened: () => void }) {
  const currentId = useChatStore((s) => s.chatId);
  const [chats, setChats] = useState<ChatSummary[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  // Reloads when the current chat gets its id: a chat started before
  // signing in is saved a moment after sign-in, maybe after this list
  // first loaded.
  useEffect(() => {
    let cancelled = false;
    api.chats
      .list()
      .then((list) => !cancelled && setChats(list))
      .catch((e) => {
        if (!cancelled) setError(e instanceof ApiError ? e.message : "Couldn't load your chats.");
      });
    return () => {
      cancelled = true;
    };
  }, [currentId]);

  const open = async (id: string) => {
    setBusyId(id);
    try {
      useChatStore.getState().open(await api.chats.get(id));
      onOpened();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Couldn't open that chat.");
      setBusyId(null);
    }
  };

  const remove = async (id: string, title: string) => {
    if (!window.confirm(`Delete "${title}"? This can't be undone.`)) return;
    setBusyId(id);
    try {
      await api.chats.remove(id);
      setChats((list) => list?.filter((c) => c.id !== id) ?? null);
      if (useChatStore.getState().chatId === id) useChatStore.getState().reset();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Couldn't delete that chat.");
    } finally {
      setBusyId(null);
    }
  };

  if (error) {
    return (
      <p role="alert" className="px-4 py-6 text-center text-sm text-destructive">
        {error}
      </p>
    );
  }

  if (chats === null) {
    return (
      <div className="flex justify-center py-10">
        <Loader2 aria-hidden className="size-5 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (chats.length === 0) {
    return (
      <p className="px-4 py-10 text-center text-sm text-muted-foreground">
        No saved chats yet. Your conversations are saved here while you&apos;re signed in.
      </p>
    );
  }

  return (
    <ul aria-label="Saved chats" className="flex flex-col divide-y">
      {chats.map((chat) => (
        <li key={chat.id} className="flex items-center gap-2 px-2 py-1">
          <button
            type="button"
            onClick={() => open(chat.id)}
            disabled={busyId !== null}
            className="flex min-h-11 min-w-0 flex-1 items-center gap-3 rounded-[8px] px-2 py-2 text-left transition-colors duration-200 hover:bg-accent"
          >
            {busyId === chat.id ? (
              <Loader2 aria-hidden className="size-4 shrink-0 animate-spin" />
            ) : (
              <MessageSquare aria-hidden className="size-4 shrink-0 text-muted-foreground" />
            )}
            <span className="min-w-0">
              <span className="block truncate font-medium">
                {chat.title}
                {chat.id === currentId && (
                  <span className="ml-2 text-xs font-normal text-primary-text">(open)</span>
                )}
              </span>
              <span className="tabular block text-xs text-muted-foreground">
                {when(chat.updated_at)} · {chat.message_count}{" "}
                {chat.message_count === 1 ? "message" : "messages"}
              </span>
            </span>
          </button>
          <Button
            variant="ghost"
            size="icon"
            aria-label={`Delete chat: ${chat.title}`}
            disabled={busyId !== null}
            onClick={() => remove(chat.id, chat.title)}
          >
            <Trash2 aria-hidden className="size-4" />
          </Button>
        </li>
      ))}
    </ul>
  );
}
