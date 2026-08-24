import fs from "node:fs";
import path from "node:path";

const root = path.resolve(import.meta.dirname, "..");
const ignoredDirectories = new Set([".git", ".next", "node_modules"]);
const markdownFiles = [];

function walk(directory) {
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    if (ignoredDirectories.has(entry.name)) continue;
    const target = path.join(directory, entry.name);
    if (entry.isDirectory()) walk(target);
    else if (entry.name.endsWith(".md")) markdownFiles.push(target);
  }
}

walk(root);

const failures = [];
const linkPattern = /!?[[^\]]*\]\(([^)]+)\)/g;

for (const file of markdownFiles) {
  const source = fs.readFileSync(file, "utf8");
  for (const match of source.matchAll(linkPattern)) {
    let href = match[1].trim();
    if (href.startsWith("<") && href.endsWith(">")) href = href.slice(1, -1);
    href = href.split(/s+["']/)[0];
    if (!href || href.startsWith("#") || /^(https?:|mailto:)/i.test(href)) continue;
    const localPath = decodeURIComponent(href.split("#")[0]);
    const resolved = path.resolve(path.dirname(file), localPath);
    if (!fs.existsSync(resolved)) {
      failures.push(`${path.relative(root, file)} -> ${href}`);
    }
  }
}

if (failures.length) {
  console.error("Broken local documentation links:");
  for (const failure of failures) console.error(`- ${failure}`);
  process.exit(1);
}

console.log(`Checked ${markdownFiles.length} Markdown files: all local links exist.`);
