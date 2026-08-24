"use client";

import { ArrowRight } from "lucide-react";
import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";

export function PublicTrackingForm() {
  const router = useRouter();
  const [code, setCode] = useState("");

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalized = code.trim().toUpperCase().replace(/[^A-Z0-9]/g, "");
    if (normalized) router.push(`/track/${normalized}`);
  }

  return (
    <form className="about-track-form" onSubmit={submit}>
      <label htmlFor="public-order-code">Код заказа</label>
      <div>
        <input
          id="public-order-code"
          autoComplete="off"
          inputMode="text"
          maxLength={16}
          onChange={(event) => setCode(event.target.value.toUpperCase())}
          placeholder="Например, MFHP73PH7K"
          required
          value={code}
        />
        <button aria-label="Открыть заказ" type="submit">
          <ArrowRight aria-hidden="true" />
        </button>
      </div>
    </form>
  );
}
