import type { Metadata } from "next";

import { DoctorView } from "@/components/doctor/doctor-view";
import { ProfessionalGate } from "@/components/professional/professional-gate";
import { SiteHeader } from "@/components/site-header";

export const metadata: Metadata = { title: "Professional" };

export default function DoctorPage() {
  return (
    <>
      <SiteHeader role="Professional" />
      <main>
        <ProfessionalGate>
          <DoctorView />
        </ProfessionalGate>
      </main>
    </>
  );
}
