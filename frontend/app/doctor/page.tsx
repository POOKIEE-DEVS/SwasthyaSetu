import type { Metadata } from "next";

import { DoctorView } from "@/components/doctor/doctor-view";
import { SiteHeader } from "@/components/site-header";

export const metadata: Metadata = { title: "Doctor" };

export default function DoctorPage() {
  return (
    <>
      <SiteHeader role="Doctor" />
      <main>
        <DoctorView />
      </main>
    </>
  );
}
