// Post-build fix for the static export's prefetch files.
//
// When a <Link> is prefetched, the Next.js client requests segment data at
// a flat name, e.g. /doctor/__next.doctor.__PAGE__.txt (every "/" in the
// segment path becomes "."). Some builds (seen on Windows) write the same
// file nested instead: /doctor/__next.doctor/__PAGE__.txt. The browser then
// gets a 404 for every prefetch: harmless (navigation falls back to loading
// the page) but noisy, and it wastes requests on slow connections.
//
// This copies each nested file to its flat name. Where the names already
// match, there is nothing nested to copy and it does nothing.

import { copyFileSync, existsSync, readdirSync, statSync } from "node:fs";
import { join, relative, sep } from "node:path";

const OUT = new URL("../out/", import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1");
let copied = 0;

function filesUnder(dir) {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name);
    return statSync(path).isDirectory() ? filesUnder(path) : [path];
  });
}

function walk(dir) {
  for (const name of readdirSync(dir)) {
    const path = join(dir, name);
    if (!statSync(path).isDirectory()) continue;
    if (name.startsWith("__next.")) {
      for (const file of filesUnder(path)) {
        const flat = join(dir, `${name}.${relative(path, file).split(sep).join(".")}`);
        if (!existsSync(flat)) {
          copyFileSync(file, flat);
          copied += 1;
        }
      }
    } else {
      walk(path);
    }
  }
}

if (existsSync(OUT)) {
  walk(OUT);
  console.log(`flatten-segments: ${copied} prefetch file(s) given flat names`);
}
