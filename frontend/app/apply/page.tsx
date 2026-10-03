import type { Metadata } from "next";

import { ApplyView } from "@/components/professional/apply-view";
import { SiteHeader } from "@/components/site-header";

export const metadata: Metadata = { title: "Get verified" };

export default function ApplyPage() {
  return (
    <>
      <SiteHeader role="Professional" />
      <main id="main" className="page-enter mx-auto w-full max-w-2xl px-4 py-8">
        <ApplyView />
      </main>
    </>
  );
}
