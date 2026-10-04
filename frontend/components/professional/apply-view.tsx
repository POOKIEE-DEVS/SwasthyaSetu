"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { BadgeCheck, Clock, Loader2, ShieldCheck } from "lucide-react";

import { DocumentInput } from "@/components/professional/document-input";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  api,
  ApiError,
  ROLE_LABEL,
  type Application,
  type DocumentKind,
  type ProfessionalRole,
} from "@/lib/api";
import { isProfessional, useAuth, useAuthStore, verifiedRole } from "@/lib/store/auth";
import { cn } from "@/lib/utils";

/** The council each registered profession gives its number from. MBBS
 * students give a doctor's recommendation instead. */
const COUNCIL: Record<Exclude<ProfessionalRole, "student">, { name: string; short: string }> = {
  doctor: { name: "Nepal Medical Council", short: "NMC" },
  pharmacist: { name: "Nepal Pharmacy Council", short: "NPC" },
  nurse: { name: "Nepal Nursing Council", short: "NNC" },
  paramedic: { name: "Nepal Health Professional Council", short: "NHPC" },
};

const ROLE_OPTIONS: ProfessionalRole[] = ["doctor", "pharmacist", "nurse", "paramedic", "student"];

type Fields = {
  full_name: string;
  phone: string;
  citizenship_number: string;
  citizenship_district: string;
  council_number: string;
  institution: string;
  recommender_name: string;
  recommender_nmc: string;
};

const EMPTY: Fields = {
  full_name: "",
  phone: "",
  citizenship_number: "",
  citizenship_district: "",
  council_number: "",
  institution: "",
  recommender_name: "",
  recommender_nmc: "",
};

function fromApplication(a: Application | null): Fields {
  if (!a) return EMPTY;
  return {
    full_name: a.full_name,
    phone: a.phone,
    citizenship_number: a.citizenship_number,
    citizenship_district: a.citizenship_district,
    council_number: a.council_number ?? "",
    institution: a.institution ?? "",
    recommender_name: a.recommender_name ?? "",
    recommender_nmc: a.recommender_nmc ?? "",
  };
}

function Field({
  label,
  value,
  onChange,
  ...props
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
} & Omit<React.ComponentProps<typeof Input>, "value" | "onChange">) {
  return (
    <label className="flex flex-col gap-1.5 text-sm font-medium">
      {label}
      <Input value={value} onChange={(event) => onChange(event.target.value)} required {...props} />
    </label>
  );
}

function ApplicationForm({
  existing,
  initialRole,
}: {
  existing: Application | null;
  initialRole: ProfessionalRole;
}) {
  const router = useRouter();
  const [role, setRole] = useState<ProfessionalRole>(existing?.role ?? initialRole);
  const [fields, setFields] = useState<Fields>(() => fromApplication(existing));
  const [files, setFiles] = useState<Partial<Record<DocumentKind, File>>>({});
  const [consent, setConsent] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const set = (name: keyof Fields) => (value: string) =>
    setFields((current) => ({ ...current, [name]: value }));
  const setFile = (kind: DocumentKind) => (file: File | null) =>
    setFiles((current) => {
      const next = { ...current };
      if (file) next[kind] = file;
      else delete next[kind];
      return next;
    });

  const needed: DocumentKind[] = [
    "citizenship_front",
    "citizenship_back",
    role === "student" ? "recommendation_letter" : "council_certificate",
  ];
  const missingDocument = needed.some((kind) => !files[kind]);

  const submit = async () => {
    setError(null);
    if (missingDocument) {
      setError("Please add all the required documents.");
      return;
    }
    const form = new FormData();
    form.set("role", role);
    form.set("consent", String(consent));
    const textFields: (keyof Fields)[] = [
      "full_name",
      "phone",
      "citizenship_number",
      "citizenship_district",
      ...(role === "student"
        ? (["institution", "recommender_name", "recommender_nmc"] as const)
        : (["council_number"] as const)),
    ];
    for (const name of textFields) form.set(name, fields[name].trim());
    for (const [kind, file] of Object.entries(files)) {
      if (kind === "council_certificate" && role === "student") continue;
      if (kind === "recommendation_letter" && role !== "student") continue;
      form.set(kind, file);
    }

    setSubmitting(true);
    try {
      await api.applications.submit(form);
      await useAuthStore.getState().load();
      router.push("/doctor/");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Couldn't submit. Please try again.");
      setSubmitting(false);
    }
  };

  return (
    <form
      className="flex flex-col gap-6"
      onSubmit={(event) => {
        event.preventDefault();
        void submit();
      }}
    >
      {existing?.status === "rejected" && existing.rejection_reason && (
        <p role="alert" className="rounded-lg border border-destructive/40 bg-destructive/5 px-3 py-2 text-sm">
          <span className="font-medium">The reviewer asked for changes:</span>{" "}
          {existing.rejection_reason}
        </p>
      )}
      {existing && (
        <p className="text-sm text-muted-foreground">
          Your details are filled in. Please add your documents again: they are replaced
          when you resubmit.
        </p>
      )}

      <fieldset className="flex flex-col gap-2">
        <legend className="mb-2 text-sm font-medium">I am a</legend>
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3" role="radiogroup">
          {ROLE_OPTIONS.map((option) => (
            <button
              key={option}
              type="button"
              role="radio"
              aria-checked={role === option}
              onClick={() => setRole(option)}
              className={cn(
                "rounded-lg border px-2 py-2.5 text-sm font-medium",
                role === option ? "border-primary bg-primary/10 text-primary-text" : "hover:bg-accent",
              )}
            >
              {ROLE_LABEL[option]}
            </button>
          ))}
        </div>
      </fieldset>

      <section className="flex flex-col gap-4">
        <h2 className="font-semibold">Your identity · तपाईंको पहिचान</h2>
        <Field
          label="Full name, as on your citizenship · पूरा नाम"
          value={fields.full_name}
          onChange={set("full_name")}
          autoComplete="name"
          maxLength={120}
        />
        <Field
          label="Phone number · फोन नम्बर"
          value={fields.phone}
          onChange={set("phone")}
          type="tel"
          autoComplete="tel"
          placeholder="98XXXXXXXX"
          maxLength={20}
        />
        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            label="Citizenship number · नागरिकता नम्बर"
            value={fields.citizenship_number}
            onChange={set("citizenship_number")}
            maxLength={40}
          />
          <Field
            label="Issuing district · जारी जिल्ला"
            value={fields.citizenship_district}
            onChange={set("citizenship_district")}
            maxLength={60}
          />
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <DocumentInput
            label="Citizenship: front"
            value={files.citizenship_front ?? null}
            onChange={setFile("citizenship_front")}
            required
          />
          <DocumentInput
            label="Citizenship: back"
            value={files.citizenship_back ?? null}
            onChange={setFile("citizenship_back")}
            required
          />
        </div>
        <DocumentInput
          label="Selfie holding your citizenship"
          hint="Helps the reviewer confirm the documents are yours."
          value={files.selfie ?? null}
          onChange={setFile("selfie")}
          allowPdf={false}
        />
      </section>

      {role === "student" ? (
        <section className="flex flex-col gap-4">
          <h2 className="font-semibold">Your studies and recommendation</h2>
          <Field
            label="Medical college"
            value={fields.institution}
            onChange={set("institution")}
            maxLength={160}
          />
          <div className="grid gap-4 sm:grid-cols-2">
            <Field
              label="Recommending doctor's name"
              value={fields.recommender_name}
              onChange={set("recommender_name")}
              maxLength={120}
            />
            <Field
              label="Recommending doctor's NMC number"
              value={fields.recommender_nmc}
              onChange={set("recommender_nmc")}
              maxLength={40}
            />
          </div>
          <DocumentInput
            label="Letter of recommendation from the doctor"
            hint="Signed by the doctor named above."
            value={files.recommendation_letter ?? null}
            onChange={setFile("recommendation_letter")}
            required
          />
        </section>
      ) : (
        <section className="flex flex-col gap-4">
          <h2 className="font-semibold">Your registration</h2>
          <Field
            label={`${COUNCIL[role].name} (${COUNCIL[role].short}) registration number`}
            value={fields.council_number}
            onChange={set("council_number")}
            maxLength={40}
          />
          <DocumentInput
            label={`${COUNCIL[role].short} registration certificate`}
            value={files.council_certificate ?? null}
            onChange={setFile("council_certificate")}
            required
          />
        </section>
      )}

      <label className="flex items-start gap-2.5 text-sm">
        <input
          type="checkbox"
          checked={consent}
          onChange={(event) => setConsent(event.target.checked)}
          className="mt-0.5 size-4 accent-[var(--primary)]"
          required
        />
        <span>
          I agree that SwasthyaSetu keeps these documents to verify my identity and
          registration. Only the reviewing admin can see them.
        </span>
      </label>

      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}

      <Button type="submit" size="lg" disabled={submitting}>
        {submitting && <Loader2 aria-hidden className="animate-spin" />}
        {existing ? "Resubmit for review" : "Submit for review"}
      </Button>
    </form>
  );
}

export function ApplyView() {
  const { user, ready } = useAuth();
  const [application, setApplication] = useState<Application | null | undefined>(undefined);
  const [editing, setEditing] = useState(false);

  useEffect(() => {
    if (!user) return;
    let cancelled = false;
    api.applications
      .mine()
      .then((a) => !cancelled && setApplication(a))
      .catch(() => !cancelled && setApplication(null));
    return () => {
      cancelled = true;
    };
  }, [user]);

  if (!ready || (user && application === undefined)) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 aria-hidden className="size-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!user) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-xl">Verification for medical professionals</CardTitle>
          <CardDescription>
            Sign in with Google first, then submit your documents for review.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button asChild size="lg" className="w-full">
            <Link href="/account/?next=/apply/">Sign in</Link>
          </Button>
        </CardContent>
      </Card>
    );
  }

  if (verifiedRole(user)) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-xl">
            <BadgeCheck aria-hidden className="size-6 text-triage-green" />
            You&apos;re verified
          </CardTitle>
          <CardDescription>You can now see waiting patients and take their calls.</CardDescription>
        </CardHeader>
        <CardContent>
          <Button asChild size="lg" className="w-full">
            <Link href="/doctor/">Go to the professional dashboard</Link>
          </Button>
        </CardContent>
      </Card>
    );
  }

  if (application?.status === "pending" && !editing) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-xl">
            <Clock aria-hidden className="size-6 text-triage-yellow" />
            Under review
          </CardTitle>
          <CardDescription>
            Your {ROLE_LABEL[application.role]} application was submitted on{" "}
            {new Date(application.submitted_at * 1000).toLocaleString()}. The admin checks
            every document by hand. You&apos;ll be able to take calls once it is approved.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          <Button asChild>
            <Link href="/doctor/">Check status</Link>
          </Button>
          <Button variant="outline" onClick={() => setEditing(true)}>
            Update my application
          </Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-semibold">
          <ShieldCheck aria-hidden className="size-6 text-primary-text" />
          Get verified
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Doctors, pharmacists, nurses, paramedics and MBBS students can take patient calls
          once an admin has checked their documents.
        </p>
      </div>
      <ApplicationForm
        existing={application ?? null}
        initialRole={isProfessional(user.role) ? user.role : "doctor"}
      />
    </div>
  );
}
