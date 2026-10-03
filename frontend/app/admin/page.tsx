import type { Metadata } from "next";

import { AdminView } from "@/components/admin/admin-view";
import { SiteHeader } from "@/components/site-header";

export const metadata: Metadata = { title: "Admin review" };

export default function AdminPage() {
  return (
    <>
      <SiteHeader role="Admin" />
      <main id="main" className="mx-auto w-full max-w-4xl px-4 py-8">
        <AdminView />
      </main>
    </>
  );
}
