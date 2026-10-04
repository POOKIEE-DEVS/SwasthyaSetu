"use client";

import { useEffect, useState } from "react";
import { HeartHandshake, Loader2 } from "lucide-react";

import { api, type HelpSummary } from "@/lib/api";

function formatDate(timestamp: number): string {
  return new Date(timestamp * 1000).toLocaleDateString(undefined, {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return "under a minute";
  const minutes = Math.round(seconds / 60);
  return `${minutes} min`;
}

/**
 * "You've helped N people", and who: the professional's own record, from
 * the server (each professional only ever sees theirs). Loaded when the
 * page appears, so it is up to date as soon as a call ends and the
 * professional comes back here.
 */
export function HelpRecord() {
  const [record, setRecord] = useState<HelpSummary | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let cancelled = false;
    api
      .helped()
      .then((data) => !cancelled && setRecord(data))
      .catch(() => !cancelled && setFailed(true));
    return () => {
      cancelled = true;
    };
  }, []);

  if (failed) return null;

  return (
    <section
      aria-labelledby="help-record-title"
      data-testid="help-record"
      className="mt-8 rounded-[12px] border bg-card p-5 sm:p-6"
    >
      <div className="flex items-start gap-3">
        <HeartHandshake aria-hidden className="mt-1 size-5 shrink-0 text-primary-text" />
        <div className="min-w-0 flex-1">
          <h2 id="help-record-title" className="text-sm font-medium text-muted-foreground">
            Your help record
          </h2>
          {record === null ? (
            <Loader2 aria-label="Loading" className="mt-2 size-5 animate-spin text-muted-foreground" />
          ) : (
            <p className="mt-1 font-display text-3xl">
              You&apos;ve helped{" "}
              <span className="tabular" data-testid="help-count">
                {record.count}
              </span>{" "}
              {record.count === 1 ? "person" : "people"}
            </p>
          )}
        </div>
      </div>

      {record && record.people.length === 0 && (
        <p className="mt-3 text-sm text-muted-foreground">
          Everyone you help on a call will appear here.
        </p>
      )}

      {record && record.people.length > 0 && (
        <ul aria-label="People you have helped" className="mt-4 divide-y border-t">
          {record.people.map((person, i) => (
            <li
              key={`${person.started_at}-${i}`}
              className="flex items-baseline justify-between gap-4 py-3 text-sm"
            >
              <span className="min-w-0 truncate font-medium">{person.patient_name}</span>
              <span className="shrink-0 text-muted-foreground tabular">
                {formatDate(person.started_at)} · {formatDuration(person.duration_seconds)}
              </span>
            </li>
          ))}
        </ul>
      )}
      {record && record.count > record.people.length && (
        <p className="mt-2 text-xs text-muted-foreground">
          Showing the {record.people.length} most recent.
        </p>
      )}
    </section>
  );
}
