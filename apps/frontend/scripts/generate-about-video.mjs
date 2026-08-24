import { chromium } from "@playwright/test";
import { readFile, writeFile } from "node:fs/promises";
import { extname } from "node:path";

const [input, output, mode = "hero"] = process.argv.slice(2);
if (!input || !output) {
  throw new Error("Usage: node scripts/generate-about-video.mjs INPUT OUTPUT [hero|macro]");
}

const mime = extname(input).toLowerCase() === ".jpg" ? "image/jpeg" : "image/png";
const imageData = (await readFile(input)).toString("base64");
const browser = await chromium.launch({ headless: true });

try {
  const page = await browser.newPage({ viewport: { width: 1280, height: 720 } });
  const encoded = await page.evaluate(async ({ imageData, mime, mode }) => {
    const canvas = document.createElement("canvas");
    canvas.width = 1280;
    canvas.height = 720;
    const context = canvas.getContext("2d");
    if (!context) throw new Error("Canvas is unavailable");

    const image = new Image();
    image.src = `data:${mime};base64,${imageData}`;
    await image.decode();

    const stream = canvas.captureStream(24);
    const supported = [
      "video/webm;codecs=vp9",
      "video/webm;codecs=vp8",
      "video/webm",
    ].find((candidate) => MediaRecorder.isTypeSupported(candidate));
    if (!supported) throw new Error("No supported WebM codec");

    const chunks = [];
    const recorder = new MediaRecorder(stream, {
      mimeType: supported,
      videoBitsPerSecond: 1_700_000,
    });
    recorder.ondataavailable = (event) => {
      if (event.data.size) chunks.push(event.data);
    };
    const stopped = new Promise((resolve) => {
      recorder.onstop = resolve;
    });
    recorder.start(500);

    const duration = mode === "hero" ? 9000 : 7600;
    const started = performance.now();
    const particles = Array.from({ length: 18 }, (_, index) => ({
      x: (index * 83) % 1280,
      y: (index * 127) % 720,
      radius: 0.7 + (index % 3) * 0.45,
      speed: 5 + (index % 5) * 2,
      alpha: 0.08 + (index % 4) * 0.025,
    }));

    await new Promise((resolve) => {
      function frame(now) {
        const elapsed = now - started;
        const progress = Math.min(1, elapsed / duration);
        const eased = (1 - Math.cos(progress * Math.PI)) / 2;
        const coverScale = Math.max(1280 / image.width, 720 / image.height);
        const zoom = coverScale * (mode === "hero" ? 1.03 + eased * 0.075 : 1.07 + eased * 0.1);
        const width = image.width * zoom;
        const height = image.height * zoom;
        const driftX = mode === "hero" ? eased * -34 : eased * 28;
        const driftY = Math.sin(progress * Math.PI) * -13;

        context.clearRect(0, 0, 1280, 720);
        context.drawImage(
          image,
          (1280 - width) / 2 + driftX,
          (720 - height) / 2 + driftY,
          width,
          height,
        );

        const vignette = context.createRadialGradient(720, 340, 120, 640, 360, 760);
        vignette.addColorStop(0, "rgba(9,11,10,0)");
        vignette.addColorStop(1, "rgba(6,8,7,.5)");
        context.fillStyle = vignette;
        context.fillRect(0, 0, 1280, 720);

        const glowX = mode === "hero" ? 940 + Math.sin(progress * Math.PI * 2) * 55 : 420 + Math.sin(progress * Math.PI * 2) * 70;
        const glow = context.createRadialGradient(glowX, 500, 10, glowX, 500, 310);
        glow.addColorStop(0, "rgba(143,120,255,.13)");
        glow.addColorStop(1, "rgba(143,120,255,0)");
        context.fillStyle = glow;
        context.fillRect(0, 0, 1280, 720);

        for (const particle of particles) {
          const y = (particle.y - elapsed / 1000 * particle.speed + 760) % 760 - 20;
          context.beginPath();
          context.fillStyle = `rgba(218,255,112,${particle.alpha})`;
          context.arc(particle.x + Math.sin(elapsed / 1400 + particle.x) * 5, y, particle.radius, 0, Math.PI * 2);
          context.fill();
        }

        if (elapsed < duration) {
          requestAnimationFrame(frame);
        } else {
          recorder.stop();
          resolve();
        }
      }
      requestAnimationFrame(frame);
    });

    await stopped;
    const blob = new Blob(chunks, { type: supported });
    const bytes = new Uint8Array(await blob.arrayBuffer());
    let binary = "";
    for (let offset = 0; offset < bytes.length; offset += 0x8000) {
      binary += String.fromCharCode(...bytes.subarray(offset, offset + 0x8000));
    }
    return btoa(binary);
  }, { imageData, mime, mode });

  await writeFile(output, Buffer.from(encoded, "base64"));
} finally {
  await browser.close();
}
