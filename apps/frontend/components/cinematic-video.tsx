"use client";

import { useEffect, useRef, useState } from "react";

type CinematicVideoProps = {
  src: string;
  poster: string;
  className?: string;
  controls?: boolean;
  label: string;
};

export function CinematicVideo({ src, poster, className, controls = false, label }: CinematicVideoProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    const video = videoRef.current;
    const media = window.matchMedia("(prefers-reduced-motion: reduce)");

    const syncMotionPreference = () => {
      if (!video) return;
      if (media.matches) video.pause();
      else void video.play().catch(() => undefined);
    };

    syncMotionPreference();
    media.addEventListener("change", syncMotionPreference);
    return () => media.removeEventListener("change", syncMotionPreference);
  }, []);

  if (failed) {
    return (
      <span
        aria-label={label}
        className={className}
        role="img"
        style={{ backgroundImage: `url(${poster})`, backgroundPosition: "center", backgroundSize: "cover" }}
      />
    );
  }

  return (
    <video
      ref={videoRef}
      aria-label={label}
      autoPlay
      className={className}
      controls={controls}
      loop
      muted
      playsInline
      poster={poster}
      preload="metadata"
      onError={() => setFailed(true)}
    >
      <source src={src} type="video/webm" />
    </video>
  );
}
