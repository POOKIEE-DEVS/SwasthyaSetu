"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { Check, ExternalLink, FileText, Loader2, RefreshCw, ShieldAlert, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import {
  api,
  ApiError,
  ROLE_LABEL,
  type AdminApplication,
  type DocumentKind,
  type VerificationStatus,
} from "@/lib/api";
import { useAuth } from "@/lib/store/auth";
import { cn } from "@/lib/utils";

const DOCUMENT_LABEL: Record<DocumentKind, string> = {
  citizenship_front: "Citizenship (front)",
  citizenship_back: "Citizenship (back)",
  council_certificate: "Council certificate",
  recommendation_letter: "Letter of recommendation",
  selfie: "Selfie with citizenship",
};

const TABS: { status: VerificationStatus; label: string }[] = [
  { status: "pending", label: "Waiting for review" },
  { status: "approved", label: "Approved" },
  { status: "rejected", label: "Rejected" },
];

function when(timestamp: number): string {
  return new Date(timestamp * 1000).toLocaleString(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  });
}

function Detail({ label, value }: { label: string; value: string | null }) {
  if (!value) return null;
  return (
    <div>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="font-medium break-words">{value}</dd>
    </div>
  );
}

function ApplicationCard({
  application,
  onDecided,
}: {
  application: AdminApplication;
  onDecided: () => void;
}) {
  const [rejecting, setRejecting] = useState(false);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const a = application;
  const councilLabel =
    a.role === "doctor" ? "NMC number" : a.role === "pharmacist" ? "Pharmacy Council number" : null;

  const act = async (action: () => Promise<unknown>) => {
    setBusy(true);
    setError(null);
    try {
      await action();
      onDecided();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "That didn't work. Please try again.");
      setBusy(false);
    }
  };

  return (
    <Card data-testid="application">
      <CardHeader className="gap-1">
        <CardTitle className="flex flex-wrap items-center gap-2">
          {a.full_name}
          <span className="rounded-full bg-secondary px-2 py-0.5 text-xs font-medium">
            {ROLE_LABEL[a.role]}
          </span>
        </CardTitle>
        <CardDescription>
          {a.user_email} · submitted {when(a.submitted_at)}
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-5">
        <dl className="grid gap-3 text-sm sm:grid-cols-2">
          <Detail label="Phone" value={a.phone} />
          <Detail
            label="Citizenship number (district)"
            value={`${a.citizenship_number} (${a.citizenship_district})`}
          />
          {councilLabel && <Detail label={councilLabel} value={a.council_number} />}
          <Detail label="Medical college" value={a.institution} />
          <Detail
            label="Recommended by (NMC number)"
            value={a.recommender_name ? `${a.recommender_name} (${a.recommender_nmc})` : null}
          />
          <Detail label="Google account name" value={a.user_name} />
        </dl>

        <div>
          <p className="mb-2 text-sm font-medium">Documents</p>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
            {a.documents.map((doc) => {
              const url = api.admin.documentUrl(doc.id);
              const isImage = doc.content_type.startsWith("image/");
              return (
                <a
                  key={doc.id}
                  href={url}
                  target="_blank"
                  rel="noreferrer"
                  className="group flex flex-col gap-1.5 rounded-[14px] border p-2 transition-colors hover:border-foreground/30"
                >
                  {isImage ? (
                    // Private, admin-only image served by our API; next/image
                    // would add nothing in a static export.
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={url}
                      alt={DOCUMENT_LABEL[doc.kind]}
                      width={400}
                      height={300}
                      loading="lazy"
                      className="aspect-[4/3] w-full rounded-[10px] bg-muted object-cover"
                    />
                  ) : (
                    <span className="flex aspect-[4/3] w-full items-center justify-center rounded-[10px] bg-muted">
                      <FileText aria-hidden className="size-8 text-muted-foreground" />
                    </span>
                  )}
                  <span className="flex items-center gap-1 text-xs">
                    {DOCUMENT_LABEL[doc.kind]}
                    <ExternalLink aria-hidden className="size-3 opacity-60" />
                  </span>
                </a>
              );
            })}
          </div>
        </div>

        {a.history.length > 0 && (
          <ol className="flex flex-col gap-1 border-l pl-3 text-xs text-muted-foreground">
            {a.history.map((event, i) => (
              <li key={i}>
                {when(event.at)}:{" "}
                <span className="font-medium capitalize">{event.action}</span> by{" "}
                {event.actor_email}
                {event.reason && <> · &ldquo;{event.reason}&rdquo;</>}
              </li>
            ))}
          </ol>
        )}

        {a.status === "rejected" && a.rejection_reason && (
          <p className="rounded-md bg-destructive/5 px-3 py-2 text-sm">
            Rejected: {a.rejection_reason}
          </p>
        )}

        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}

        {rejecting ? (
          <form
            className="flex flex-col gap-2"
            onSubmit={(event) => {
              event.preventDefault();
              void act(() => api.admin.reject(a.id, reason.trim()));
            }}
          >
            <label className="flex flex-col gap-1.5 text-sm font-medium">
              Reason (the applicant will see this)
              <Textarea
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                rows={2}
                minLength={3}
                maxLength={500}
                required
                autoFocus
                placeholder="e.g. The citizenship photo is blurry. Please upload a clearer one."
              />
            </label>
            <div className="flex gap-2">
              <Button type="submit" variant="destructive" disabled={busy || reason.trim().length < 3}>
                {busy && <Loader2 aria-hidden className="animate-spin" />}
                Confirm rejection
              </Button>
              <Button type="button" variant="ghost" onClick={() => setRejecting(false)}>
                Cancel
              </Button>
            </div>
          </form>
        ) : (
          <div className="flex flex-wrap gap-2">
            {a.status !== "approved" && (
              <Button onClick={() => void act(() => api.admin.approve(a.id))} disabled={busy}>
                {busy ? <Loader2 aria-hidden className="animate-spin" /> : <Check aria-hidden />}
                Approve
              </Button>
            )}
            {a.status !== "rejected" && (
              <Button variant="outline" onClick={() => setRejecting(true)} disabled={busy}>
                <X aria-hidden />
                {a.status === "approved" ? "Revoke approval" : "Reject"}
              </Button>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export function AdminView() {
  const { user, ready } = useAuth();
  const [tab, setTab] = useState<VerificationStatus>("pending");
  const [items, setItems] = useState<AdminApplication[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [version, setVersion] = useState(0);
  const reload = useCallback(() => setVersion((v) => v + 1), []);
  const isAdmin = Boolean(user?.is_admin);

  useEffect(() => {
    if (!isAdmin) return;
    let cancelled = false;
    api.admin
      .list(tab)
      .then((list) => {
        if (cancelled) return;
        setItems(list);
        setError(null);
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof ApiError ? e.message : "Couldn't load applications.");
      });
    return () => {
      cancelled = true;
    };
  }, [isAdmin, tab, version]);

  if (!ready) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 aria-hidden className="size-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!user || !user.is_admin) {
    return (
      <Card className="mx-auto max-w-md">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ShieldAlert aria-hidden className="size-5" />
            Admins only
          </CardTitle>
          <CardDescription>
            {user
              ? `${user.email} isn't an admin account.`
              : "Sign in with the admin Google account to review applications."}
          </CardDescription>
        </CardHeader>
        {!user && (
          <CardContent>
            <Button asChild className="w-full">
              <Link href="/account/?next=/admin/">Sign in</Link>
            </Button>
          </CardContent>
        )}
      </Card>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight">Verification review</h1>
          <p className="text-sm text-muted-foreground">
            Check each document, look the council number up on the council&apos;s register,
            then approve or reject.
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={reload}>
          <RefreshCw aria-hidden />
          Refresh
        </Button>
      </div>

      <div role="tablist" aria-label="Application status" className="flex gap-1 rounded-full bg-muted p-1">
        {TABS.map(({ status, label }) => (
          <button
            key={status}
            role="tab"
            type="button"
            aria-selected={tab === status}
            onClick={() => {
              setItems(null);
              setTab(status);
            }}
            className={cn(
              "flex-1 rounded-full px-3 py-2 text-sm font-semibold transition-colors",
              tab === status ? "bg-card shadow-soft" : "text-muted-foreground hover:text-foreground",
            )}
          >
            {label}
          </button>
        ))}
      </div>

      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}

      {items === null ? (
        <div className="flex justify-center py-10">
          <Loader2 aria-hidden className="size-5 animate-spin text-muted-foreground" />
        </div>
      ) : items.length === 0 ? (
        <p className="rounded-[20px] border border-dashed p-12 text-center text-muted-foreground">
          {tab === "pending" ? "No applications waiting. All caught up." : "Nothing here yet."}
        </p>
      ) : (
        <div className="flex flex-col gap-4">
          {items.map((application) => (
            <ApplicationCard key={application.id} application={application} onDecided={reload} />
          ))}
        </div>
      )}
    </div>
  );
}
