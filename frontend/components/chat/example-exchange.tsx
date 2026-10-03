"use client";

import { Video } from "lucide-react";

import { BrandMark } from "@/components/brand";
import { useCopy } from "@/lib/i18n";
import { cn } from "@/lib/utils";

/**
 * A worked example of the first-aid assistant: one question, the kind of
 * reply it gives, and what can come next. Drawn like the real chat, but
 * labelled as an example and never a real message (no data-role), so it
 * can't be mistaken for, or counted as, a reply.
 */
export function ExampleExchange({ className }: { className?: string }) {
  const { t } = useCopy();
  return (
    <figure aria-label={t.chat.exampleLabel} className={cn("flex flex-col gap-4", className)}>
      <figcaption className="text-xs font-medium uppercase tracking-[0.08em] text-muted-foreground">
        {t.chat.exampleLabel}
      </figcaption>
      <p className="ml-auto max-w-[85%] rounded-[12px] rounded-br-[4px] bg-deep px-4 py-2.5 text-[15px] leading-relaxed text-deep-foreground">
        {t.chat.exampleUser}
      </p>
      <div className="flex max-w-[92%] gap-2.5">
        <BrandMark className="mt-0.5 size-7" />
        <p className="rounded-[12px] rounded-tl-[4px] bg-muted px-4 py-2.5 text-[15px] leading-relaxed">
          {t.chat.exampleReply}
        </p>
      </div>
      <p className="flex items-start gap-2 text-sm text-muted-foreground">
        <Video aria-hidden className="mt-0.5 size-4 shrink-0" />
        {t.chat.exampleNext}
      </p>
    </figure>
  );
}
