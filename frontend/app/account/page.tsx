import type { Metadata } from "next";

import { AccountView } from "@/components/account/account-view";
import { SiteHeader } from "@/components/site-header";

export const metadata: Metadata = { title: "Account" };

export default function AccountPage() {
  return (
    <>
      <SiteHeader role="Account" />
      <main className="mx-auto w-full max-w-xl px-4 py-10">
        <AccountView />
      </main>
    </>
  );
}
