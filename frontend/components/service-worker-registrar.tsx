"use client";

import { useEffect } from "react";

/**
 * Registers the service worker that makes this a Progressive Web App.
 *
 * Registration is deferred until after `load` so that fetching and parsing
 * the worker never competes with the first paint on a slow connection.
 * Development is skipped entirely: a cached shell is exactly what you do not
 * want while editing.
 */
export function ServiceWorkerRegistrar() {
  useEffect(() => {
    if (process.env.NODE_ENV !== "production") return;
    if (!("serviceWorker" in navigator)) return;

    const register = () => {
      navigator.serviceWorker.register("/sw.js").catch((error) => {
        // A failed registration costs offline support, not the app itself,
        // so it is logged rather than surfaced to the user.
        console.error("Service worker registration failed", error);
      });
    };

    if (document.readyState === "complete") {
      register();
      return;
    }

    window.addEventListener("load", register);
    return () => window.removeEventListener("load", register);
  }, []);

  return null;
}
