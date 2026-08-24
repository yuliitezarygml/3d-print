import type { MetadataRoute } from "next";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "PrintForge — управление 3D-печатью",
    short_name: "PrintForge",
    description: "Заказы, модели, производство и себестоимость 3D-печати.",
    start_url: "/dashboard",
    display: "standalone",
    background_color: "#0d100f",
    theme_color: "#8f78ff",
    lang: "ru",
    icons: [
      { src: "/icons/printforge.svg", sizes: "any", type: "image/svg+xml", purpose: "any" },
      { src: "/icons/printforge.svg", sizes: "any", type: "image/svg+xml", purpose: "maskable" },
    ],
  };
}
