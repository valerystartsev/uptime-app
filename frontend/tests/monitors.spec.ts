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
