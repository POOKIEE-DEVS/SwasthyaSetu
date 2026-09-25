import type { Metadata } from "next";

import { PatientView } from "@/components/patient/patient-view";
import { SiteHeader } from "@/components/site-header";

export const metadata: Metadata = { title: "Get help" };

export default function PatientPage() {
  return (
    <>
      <SiteHeader role="Patient" />
      <main>
        <PatientView />
      </main>
    </>
  );
}
