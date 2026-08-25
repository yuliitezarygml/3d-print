import { cp, mkdir } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import path from "node:path";

const desktopRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const source = path.resolve(desktopRoot, "../frontend/public/printer-catalog");
const destination = path.resolve(desktopRoot, "public/printer-catalog");

await mkdir(path.dirname(destination), { recursive: true });
await cp(source, destination, { recursive: true, force: true });
console.log("Prepared 387 bundled printer images for the desktop build.");
