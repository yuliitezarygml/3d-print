import { expect, test } from "@playwright/test";

test("публичная страница знакомит клиента с мастерской", async ({ page }) => {
  const mediaResponses: number[] = [];
  page.on("response", (response) => {
    if (response.url().includes("/media/about-")) mediaResponses.push(response.status());
  });

  await page.goto("/about");
  await expect(page.getByRole("heading", { name: /Идея становится/ })).toBeVisible();
  await expect(page.getByRole("link", { name: /Рассчитать печать/ })).toHaveAttribute("href", "/request");
  await expect(page.getByRole("heading", { name: /четыре понятных шага/ })).toBeVisible();
  await expect(page.getByText("Bambu Lab X1 Carbon")).toBeVisible();
  await expect(page.getByLabel("Код заказа")).toBeVisible();
  await expect(page.locator("video")).toHaveCount(3);
  await expect.poll(() => mediaResponses.some((status) => status >= 200 && status < 400)).toBeTruthy();

  await page.getByLabel("Код заказа").fill("MFHP73PH7K");
  await page.getByRole("button", { name: "Открыть заказ" }).click();
  await expect(page).toHaveURL(/\/track\/MFHP73PH7K$/);
});

test("администратор входит и открывает основные рабочие разделы", async ({ page }) => {
  await page.goto("/login");
  await page.getByRole("button", { name: /Войти/ }).click();
  await expect(page).toHaveURL(/\/dashboard/);
  await expect(page.getByRole("heading", { name: "Добрый день, Administrator" })).toBeVisible();
  await page.goto("/calendar");
  await expect(page.getByRole("heading", { name: "Календарь печати" })).toBeVisible();
  await page.goto("/orders");
  await expect(page.getByRole("heading", { name: "Заказы" })).toBeVisible();
});

test("клиент отправляет 3D-модель и получает код отслеживания", async ({ page }) => {
  await page.goto("/request");
  await page.getByLabel("Ваше имя").fill("Playwright Customer");
  await page.getByLabel("Email").fill(`playwright-${Date.now()}@example.test`);
  await page.getByLabel("Выберите 3D-файл").setInputFiles({
    name: "quality-check.stl",
    mimeType: "model/stl",
    buffer: Buffer.from("solid test\nfacet normal 0 0 1\nouter loop\nvertex 0 0 0\nvertex 10 0 0\nvertex 0 10 0\nendloop\nendfacet\nendsolid test\n"),
  });
  await page.getByRole("button", { name: "Отправить модель в мастерскую" }).click();
  await expect(page.getByRole("heading", { name: "Модель уже в мастерской" })).toBeVisible();
  const code = (await page.locator(".tracking-code").textContent())?.trim() ?? "";
  expect(code).toMatch(/^[23456789A-HJ-NP-Z]{10}$/);
  await page.getByRole("link", { name: /Открыть заказ/ }).click();
  await expect(page).toHaveURL(new RegExp(`/track/${code}$`));
  await expect(page.getByText("Заявка получена")).toBeVisible();
  await expect(page.getByText("quality-check")).toBeVisible();
});

test("публичные страницы удобны на мобильном экране", async ({ page }, testInfo) => {
  test.skip(!testInfo.project.name.includes("mobile"), "mobile-only assertion");
  await page.goto("/request");
  await expect(page.getByRole("heading", { name: /Превратим вашу 3D-модель/ })).toBeVisible();
  await expect(page.locator(".request-form")).toBeInViewport();
});
