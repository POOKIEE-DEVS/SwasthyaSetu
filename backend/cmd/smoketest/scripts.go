package main

// JavaScript that runs inside each page.

// finders looks elements up the way a person (or Playwright) does: by their
// label, their role and accessible name, their text or their data-testid.
// Each finder returns an array; root is an element, undefined for the whole
// document, or null for "nothing" (an empty parent).
const finders = `(() => {
  if (window.__smoke) return;
  const norm = (s) => String(s ?? "").replace(/\s+/g, " ").trim();
  const low = (s) => norm(s).toLowerCase();
  const visible = (el) => {
    if (!el || !el.isConnected) return false;
    const style = getComputedStyle(el);
    if (style.visibility === "hidden" || style.display === "none") return false;
    return el.getClientRects().length > 0;
  };
  const accessible = (el) => visible(el) && !el.closest('[aria-hidden="true"]');
  const accessibleName = (el) => {
    const aria = el.getAttribute("aria-label");
    if (aria) return norm(aria);
    const by = el.getAttribute("aria-labelledby");
    if (by) {
      return norm(by.split(/\s+/).map((id) => document.getElementById(id)?.innerText ?? "").join(" "));
    }
    if (el.labels && el.labels.length) return norm([...el.labels].map((l) => l.innerText).join(" "));
    if (el.tagName === "IMG") return norm(el.alt);
    if (el.tagName === "INPUT") return norm(el.value);
    return norm(el.innerText ?? el.textContent) || norm(el.title);
  };
  const ROLES = {
    button: 'button, [role="button"], input[type="button"], input[type="submit"]',
    link: 'a[href], [role="link"]',
    heading: 'h1, h2, h3, h4, h5, h6, [role="heading"]',
    checkbox: 'input[type="checkbox"], [role="checkbox"]',
    listitem: 'li, [role="listitem"]',
    list: 'ul, ol, [role="list"]',
    img: 'img:not([alt=""]), [role="img"]',
  };
  const nameMatches = (name, how, want) => {
    if (how === "exact") return name === want;
    if (how === "prefix") return name.toLowerCase().startsWith(want.toLowerCase());
    return name.toLowerCase().includes(want.toLowerCase());
  };
  const all = (root, selector) =>
    root === null ? [] : [...(root === undefined ? document : root).querySelectorAll(selector)];
  window.__smoke = {
    norm,
    visible,
    role: (root, role, how, want) =>
      all(root, ROLES[role]).filter(
        (el) => accessible(el) && (want == null || nameMatches(accessibleName(el), how, want)),
      ),
    label: (root, text) => {
      const t = text.toLowerCase();
      const found = [];
      for (const l of all(root, "label")) {
        if (low(l.innerText).includes(t) && l.control) found.push(l.control);
      }
      for (const el of all(root, "[aria-label]")) {
        if (low(el.getAttribute("aria-label")).includes(t)) found.push(el);
      }
      return [...new Set(found)];
    },
    // The innermost elements whose rendered text contains the text.
    text: (root, text) => {
      const t = text.toLowerCase();
      const hits = all(root === undefined ? document.body : root, "*").filter((el) =>
        low(el.innerText).includes(t),
      );
      return hits.filter((el) => ![...el.children].some((c) => low(c.innerText).includes(t)));
    },
    testid: (root, id) => all(root, '[data-testid="' + id + '"]'),
    css: (root, selector) => all(root, selector),
    has: (els, text) => els.filter((el) => low(el.textContent).includes(text.toLowerCase())),
  };
})();`

// pcSpy records every RTCPeerConnection the page creates, and media events,
// so a failed call can be diagnosed from outside without debug code in the
// app.
const pcSpy = `(() => {
  const Original = window.RTCPeerConnection;
  if (!Original) return;
  window.__pcs = [];
  window.RTCPeerConnection = function (...args) {
    const pc = new Original(...args);
    window.__pcs.push(pc);
    return pc;
  };
  window.RTCPeerConnection.prototype = Original.prototype;
})();
(() => {
  window.__media = [];
  const t0 = performance.now();
  const log = (what) => window.__media.push(Math.round(performance.now() - t0) + "ms " + what);
  const md = navigator.mediaDevices;
  if (!md) return;
  const gum = md.getUserMedia.bind(md);
  md.getUserMedia = async (c) => {
    log("getUserMedia(" + JSON.stringify(Object.keys(c || {})) + ")");
    const stream = await gum(c);
    for (const t of stream.getTracks()) {
      log("  got " + t.kind + " " + t.id.slice(0, 6));
      t.addEventListener("ended", () => log("ended " + t.kind + " " + t.id.slice(0, 6)));
    }
    return stream;
  };
  const stop = MediaStreamTrack.prototype.stop;
  MediaStreamTrack.prototype.stop = function () {
    const caller = (new Error().stack || "").split(String.fromCharCode(10)).slice(2, 4).join(" | ").trim();
    log("stop() " + this.kind + " " + this.id.slice(0, 6) + " <- " + caller);
    return stop.call(this);
  };
})();`

// The remote video (the first <video>) is playing.
const remotePlaying = `(() => {
  const v = document.querySelectorAll("video")[0];
  return !!v && !!v.srcObject && v.videoWidth > 0 && v.readyState >= 2;
})()`

const remoteTracks = `(() => {
  const s = document.querySelectorAll("video")[0].srcObject;
  return s ? { audio: s.getAudioTracks().length, video: s.getVideoTracks().length } : null;
})()`

const streamID = `document.querySelectorAll("video")[0]?.srcObject?.id ?? ""`

// callDiagnostics reports connection, track and RTP counters for every peer
// connection and video, as one line of text each.
const callDiagnostics = `(async () => {
  const lines = [];
  let i = 0;
  for (const pc of window.__pcs ?? []) {
    const rtp = [];
    try {
      (await pc.getStats()).forEach((r) => {
        if (r.type === "inbound-rtp" || r.type === "outbound-rtp") {
          rtp.push(r.type + "/" + r.kind + " packets=" + (r.packetsReceived ?? r.packetsSent)
            + " frames=" + (r.framesDecoded ?? r.framesEncoded ?? "-"));
        }
      });
    } catch (e) { rtp.push("getStats failed: " + e); }
    const receivers = pc.getReceivers().map((r) => r.track.kind + ":" + r.track.readyState
      + (r.track.muted ? ":muted" : ""));
    lines.push("peer#" + i++ + ": conn=" + pc.connectionState + " ice=" + pc.iceConnectionState
      + " sig=" + pc.signalingState + " receivers=" + receivers.join(",") + " rtp=" + rtp.join("; "));
  }
  [...document.querySelectorAll("video")].forEach((v, n) => {
    const tracks = v.srcObject ? v.srcObject.getTracks().map((t) => t.kind + ":" + t.readyState
      + (t.enabled ? "" : ":disabled") + (t.muted ? ":muted" : "")).join(",") : "none";
    lines.push("video#" + n + ": readyState=" + v.readyState + " size=" + v.videoWidth + "x"
      + v.videoHeight + " paused=" + v.paused + " tracks=" + tracks);
  });
  return lines;
})()`
