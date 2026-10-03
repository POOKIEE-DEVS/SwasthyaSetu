"use client";

import { create } from "zustand";

/**
 * Which patient screen is showing: the chat, or the "talk to a
 * professional" request form. Shared so the header and hero buttons can
 * open the request form, not only the button inside the chat.
 */
type Stage = "chat" | "request";

type PatientUiState = {
  stage: Stage;
  setStage: (stage: Stage) => void;
};

export const usePatientUi = create<PatientUiState>()((set) => ({
  stage: "chat",
  setStage: (stage) => set({ stage }),
}));

/** Open the request form and bring it into view. */
export function openTalkToProfessional(): void {
  usePatientUi.getState().setStage("request");
  window.scrollTo({ top: 0 });
}
