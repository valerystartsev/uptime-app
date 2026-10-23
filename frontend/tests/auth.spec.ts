import { expect, test } from "@playwright/test";

function uniqueEmail() {
  return `e2e-${Date.now()}-${Math.random().toString(36).slice(2)}@example.com`;
}

async function register(page: import("@playwright/test").Page, email: string) {
  await page.goto("/register");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Пароль").fill("password1");
  await page.getByRole("button", { name: "Создать аккаунт" }).click();
  await expect(page).toHaveURL("/dashboard");
  await expect(page.getByRole("navigation", { name: "Меню пользователя" })).toContainText(email);
}

test("registration creates HttpOnly session cookies and logout clears the session", async ({ page, context }) => {
  const email = uniqueEmail();
  await register(page, email);
  await expect(page.getByRole("navigation", { name: "Меню пользователя" })).toContainText(email);

  const cookies = await context.cookies();
  const accessToken = cookies.find((cookie) => cookie.name === "access_token");
  const refreshToken = cookies.find((cookie) => cookie.name === "refresh_token");
  expect(accessToken?.httpOnly).toBe(true);
  expect(refreshToken?.httpOnly).toBe(true);

  await expect(page.getByRole("button", { name: "Выйти" })).toBeVisible();
  await page.getByRole("button", { name: "Выйти" }).click();
  await expect(page).toHaveURL("/login");
  await page.goto("/dashboard");
  await expect(page).toHaveURL("/login");
});

test("login restores an existing account session", async ({ page }) => {
  const email = uniqueEmail();
  await register(page, email);
  await page.getByRole("button", { name: "Выйти" }).click();
  await expect(page).toHaveURL("/login");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Пароль").fill("password1");
  await page.getByRole("button", { name: "Войти" }).click();
  await expect(page).toHaveURL("/dashboard");
  await expect(page.getByRole("navigation", { name: "Меню пользователя" })).toContainText(email);
});

test("profile name is saved and shown after reopening the page", async ({ page }) => {
  const email = uniqueEmail();
  await register(page, email);
  await page.getByLabel("Открыть меню пользователя").click();
  await page.getByRole("link", { name: "Профиль" }).click();
  await expect(page).toHaveURL("/profile");
  await expect(page.getByText(email, { exact: true }).last()).toBeVisible();
  await page.getByRole("button", { name: "Редактировать" }).click();
  await page.getByLabel("Имя").fill("Ada Lovelace");
  await page.getByRole("button", { name: "Сохранить" }).click();
  await expect(page.getByText("Ada Lovelace", { exact: true }).last()).toBeVisible();
  await page.reload();
  await expect(page.getByText("Ada Lovelace", { exact: true }).last()).toBeVisible();
});
