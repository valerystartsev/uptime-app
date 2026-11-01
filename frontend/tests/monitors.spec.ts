import { expect, test } from "@playwright/test";

function uniqueEmail() {
  return `monitor-${Date.now()}-${Math.random().toString(36).slice(2)}@example.com`;
}

async function register(page: import("@playwright/test").Page) {
  await page.goto("/register");
  await page.getByLabel("Email").fill(uniqueEmail());
  await page.getByLabel("Пароль").fill("password1");
  await page.getByRole("button", { name: "Создать аккаунт" }).click();
  await expect(page).toHaveURL("/dashboard");
  await expect(page.getByText("Сайтов пока нет. Добавьте первый сайт.")).toBeVisible();
}

test("create monitors with custom seconds, minutes and hours", async ({ page }) => {
  await register(page);

  for (const [value, unit, label] of [
    ["7", "seconds", "Каждые 7 секунд"],
    ["10", "minutes", "Каждые 10 минут"],
    ["2", "hours", "Каждые 2 часа"],
  ]) {
    await page.getByRole("button", { name: "Добавить сайт" }).click();
    await page.getByLabel("URL сайта").fill(`https://example.com/${unit}`);
    await page.getByLabel("Частота опроса").fill(value);
    await page.getByLabel("Единица измерения").selectOption(unit);
    await page.getByRole("button", { name: "Создать", exact: true }).click();
    await expect(page.getByRole("link", { name: `https://example.com/${unit}` })).toBeVisible();
    await expect(page.getByText(label)).toBeVisible();
  }

  await page.reload();
  await expect(page.getByText("Каждые 7 секунд")).toBeVisible();
  await expect(page.getByText("Каждые 10 минут")).toBeVisible();
  await expect(page.getByText("Каждые 2 часа")).toBeVisible();

  await page.getByRole("button", { name: "Выйти" }).click();
  await expect(page).toHaveURL("/login");
  await register(page);
  await expect(page.getByRole("link", { name: "https://example.com/seconds" })).toHaveCount(0);
});

test("monitor creation rejects cross-origin and oversized requests", async ({ page }) => {
  await register(page);
  const body = { url: "https://example.com/security", interval_value: 1, interval_unit: "minutes" };
  const cookies = (await page.context().cookies()).map(({ name, value }) => `${name}=${value}`).join("; ");
  const rejected = await page.request.post("/api/monitors", {
    headers: { Cookie: cookies, Origin: "http://attacker.example.com", "Content-Type": "application/json" },
    data: body,
  });
  expect(rejected.status()).toBe(403);

  const oversized = await page.evaluate(async (body) => {
    const response = await fetch("/api/monitors", {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ ...body, padding: "x".repeat(17 * 1024) }),
    });
    return response.status;
  }, body);
  expect(oversized).toBe(400);

  const list = await page.evaluate(async () => {
    const response = await fetch("/api/monitors?page=1");
    return { status: response.status, body: await response.json() };
  });
  expect(list.status).toBe(200);
  expect(list.body.items).toEqual([]);
});

test("rejects a duplicate monitor and explains the error", async ({ page }) => {
  await register(page);
  await page.getByRole("button", { name: "Добавить сайт" }).click();
  await page.getByLabel("URL сайта").fill("https://example.com");
  await page.getByRole("button", { name: "Создать", exact: true }).click();
  await expect(page.getByRole("link", { name: "https://example.com", exact: true })).toHaveCount(1);

  await page.getByRole("button", { name: "Добавить сайт" }).click();
  await page.getByLabel("URL сайта").fill("https://EXAMPLE.com:443/");
  await page.getByRole("button", { name: "Создать", exact: true }).click();
  await expect(page.getByRole("alert").filter({ hasText: "Этот сайт уже добавлен" })).toBeVisible();
  await expect(page.getByRole("link", { name: "https://example.com", exact: true })).toHaveCount(1);
});

test("monitor list loads a second page", async ({ page }, testInfo) => {
  test.setTimeout(120_000);
  await register(page);
  const statuses = await page.evaluate(async () => {
    const results: number[] = [];
    for (let index = 0; index < 51; index++) {
      const response = await fetch("/api/monitors", {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url: `https://example.com/page-${index}`, interval_value: 1, interval_unit: "hours" }),
      });
      results.push(response.status);
    }
    return results;
  });
  expect(statuses).toEqual(Array(51).fill(201));
  await page.reload();
  await expect(page.getByRole("button", { name: "Показать ещё" })).toBeVisible();
  await page.getByRole("button", { name: "Показать ещё" }).scrollIntoViewIfNeeded();
  await page.screenshot({ path: testInfo.outputPath("monitor-pagination.png") });
  await page.getByRole("button", { name: "Показать ещё" }).click();
  await expect(page.getByRole("link", { name: "https://example.com/page-0" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Показать ещё" })).toHaveCount(0);
});
