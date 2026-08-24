#!/usr/bin/env node

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const outputJSON = path.join(projectRoot, "apps/backend/internal/http/printer_catalog.json");
const outputImages = path.join(projectRoot, "apps/frontend/public/printer-catalog");

const options = Object.fromEntries(process.argv.slice(2).map((value, index, all) => value.startsWith("--") ? [value.slice(2), all[index + 1]] : null).filter(Boolean));
const sources = [
  {
    id: "orcaslicer",
    name: "OrcaSlicer",
    root: options.orca,
    repository: "https://github.com/OrcaSlicer/OrcaSlicer",
    branch: "main",
    license: "AGPL-3.0",
  },
  {
    id: "bambustudio",
    name: "Bambu Studio",
    root: options.bambu,
    repository: "https://github.com/bambulab/BambuStudio",
    branch: "master",
    license: "AGPL-3.0",
  },
];

for (const source of sources) {
  if (!source.root) throw new Error(`Pass --${source.id === "orcaslicer" ? "orca" : "bambu"} /path/to/repository`);
  source.root = path.resolve(source.root);
  source.profiles = path.join(source.root, "resources/profiles");
  source.revision = execFileSync("git", ["-C", source.root, "rev-parse", "HEAD"], { encoding: "utf8" }).trim();
  if (!fs.existsSync(source.profiles)) throw new Error(`Profiles not found: ${source.profiles}`);
}

fs.mkdirSync(outputImages, { recursive: true });
for (const entry of fs.readdirSync(outputImages)) fs.rmSync(path.join(outputImages, entry));

const manufacturerAliases = new Map([
  ["bambulab", "Bambu Lab"], ["bambu lab", "Bambu Lab"], ["bbl", "Bambu Lab"],
  ["prusa research", "Prusa"], ["creality3d", "Creality"],
]);
const normalize = value => String(value ?? "").trim().replace(/\s+/g, " ");
const normalizedKey = value => normalize(value).toLocaleLowerCase("en-US");
const manufacturerName = value => manufacturerAliases.get(normalizedKey(value)) ?? normalize(value);
const stableHash = value => crypto.createHash("sha256").update(value).digest("hex").slice(0, 10);
const safeSlug = value => normalize(value).normalize("NFKD").replace(/[\u0300-\u036f]/g, "").toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "").slice(0, 65) || "printer";
const readJSON = filename => {
  try { return JSON.parse(fs.readFileSync(filename, "utf8")); } catch { return null; }
};

function inheritedProfile(profile, profiles, seen = new Set()) {
  if (!profile || seen.has(profile.name)) return profile ?? {};
  const nextSeen = new Set(seen).add(profile.name);
  const parent = profile.inherits ? inheritedProfile(profiles.get(profile.inherits), profiles, nextSeen) : {};
  return { ...parent, ...profile };
}

function printerDimensions(modelName, machineProfiles) {
  const variants = [...machineProfiles.values()]
    .filter(profile => profile?.type === "machine" && profile.printer_model === modelName)
    .map(profile => inheritedProfile(profile, machineProfiles));
  const preferred = variants.find(profile => (profile.nozzle_diameter ?? []).map(String).includes("0.4")) ?? variants[0];
  if (!preferred) return { buildXmm: 0, buildYmm: 0, buildZmm: 0 };
  const points = Array.isArray(preferred.printable_area) ? preferred.printable_area.map(point => {
    const match = String(point).match(/^\s*(-?\d+(?:\.\d+)?)x(-?\d+(?:\.\d+)?)\s*$/);
    return match ? [Number(match[1]), Number(match[2])] : null;
  }).filter(Boolean) : [];
  const xs = points.map(point => point[0]);
  const ys = points.map(point => point[1]);
  const height = Array.isArray(preferred.printable_height) ? preferred.printable_height[0] : preferred.printable_height;
  return {
    buildXmm: xs.length ? Math.round((Math.max(...xs) - Math.min(...xs)) * 100) / 100 : 0,
    buildYmm: ys.length ? Math.round((Math.max(...ys) - Math.min(...ys)) * 100) / 100 : 0,
    buildZmm: Number(height) || 0,
  };
}

function modelNameWithoutManufacturer(fullName, manufacturer) {
  const prefixes = [manufacturer, manufacturer.replace("Bambu Lab", "BambuLab"), "Original Prusa"];
  for (const prefix of prefixes) {
    if (normalizedKey(fullName).startsWith(normalizedKey(prefix) + " ")) return normalize(fullName.slice(prefix.length));
  }
  return normalize(fullName);
}

const catalog = new Map();
for (const source of sources) {
  for (const metaFilename of fs.readdirSync(source.profiles).filter(filename => filename.endsWith(".json")).sort()) {
    const meta = readJSON(path.join(source.profiles, metaFilename));
    if (!meta || !Array.isArray(meta.machine_model_list)) continue;
    const vendorFolder = path.join(source.profiles, path.basename(metaFilename, ".json"));
    const machineFolder = path.join(vendorFolder, "machine");
    const machineProfiles = new Map();
    if (fs.existsSync(machineFolder)) {
      for (const filename of fs.readdirSync(machineFolder).filter(filename => filename.endsWith(".json"))) {
        const profile = readJSON(path.join(machineFolder, filename));
        if (profile?.name) machineProfiles.set(profile.name, profile);
      }
    }
    for (const listed of meta.machine_model_list) {
      const profilePath = path.join(vendorFolder, listed.sub_path ?? `machine/${listed.name}.json`);
      const profile = readJSON(profilePath) ?? machineProfiles.get(listed.name);
      if (!profile || profile.type !== "machine_model") continue;
      const manufacturer = manufacturerName(meta.name || path.basename(metaFilename, ".json"));
      const fullName = normalize(profile.name || listed.name);
      const model = modelNameWithoutManufacturer(fullName, manufacturer);
      const key = `${normalizedKey(manufacturer)}:${normalizedKey(fullName)}`;
      const dimensions = printerDimensions(fullName, machineProfiles);
      const nozzles = String(profile.nozzle_diameter ?? "0.4").split(";").map(Number).filter(value => Number.isFinite(value) && value > 0);
      const imageSource = path.join(vendorFolder, `${fullName}_cover.png`);
      const assetName = `${safeSlug(`${manufacturer}-${model}`)}-${stableHash(key)}.png`;
      const entry = {
        key: stableHash(key),
        manufacturer,
        model,
        fullName,
        modelId: normalize(profile.model_id),
        family: normalize(profile.family),
        technology: normalize(profile.machine_tech || "FFF"),
        nozzleDiameters: nozzles.length ? nozzles : [0.4],
        ...dimensions,
        imageUrl: fs.existsSync(imageSource) ? `/printer-catalog/${assetName}` : null,
        defaultMaterials: String(profile.default_materials ?? "").split(";").map(normalize).filter(Boolean).slice(0, 30),
        profileUrl: `${source.repository}/blob/${source.revision}/resources/profiles/${encodeURI(path.relative(source.profiles, profilePath)).replaceAll("#", "%23")}`,
        sources: [{ id: source.id, name: source.name, repository: source.repository, revision: source.revision, license: source.license }],
      };
      const current = catalog.get(key);
      if (current) {
        entry.sources = [...current.sources, ...entry.sources.filter(item => !current.sources.some(existing => existing.id === item.id))];
        const preferIncoming = source.id === "bambustudio" && manufacturer === "Bambu Lab";
        if (!preferIncoming) {
          current.sources = entry.sources;
          continue;
        }
      }
      catalog.set(key, entry);
      if (entry.imageUrl) fs.copyFileSync(imageSource, path.join(outputImages, assetName));
    }
  }
}

const models = [...catalog.values()].sort((a, b) => a.manufacturer.localeCompare(b.manufacturer) || a.model.localeCompare(b.model));
const payload = {
  generatedAt: new Date().toISOString(),
  total: models.length,
  sources: sources.map(({ id, name, repository, revision, license }) => ({ id, name, repository, revision, license })),
  models,
};
fs.writeFileSync(outputJSON, JSON.stringify(payload));
console.log(JSON.stringify({ outputJSON, outputImages, total: models.length, withImages: models.filter(model => model.imageUrl).length, sources: payload.sources }, null, 2));
