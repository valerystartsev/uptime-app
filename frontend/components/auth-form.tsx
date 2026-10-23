"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";

type AuthMode = "login" | "register";

type AuthFormProps = { mode: AuthMode };
type ApiError = { error?: unknown };

const content = {
  login: { title: "Войти в аккаунт", submit: "Войти", prompt: "Нет аккаунта?", linkLabel: "Зарегистрироваться", linkHref: "/register" },
  register: { title: "Создать аккаунт", submit: "Создать аккаунт", prompt: "Уже есть аккаунт?", linkLabel: "Войти", linkHref: "/login" },
};

export function AuthForm({ mode }: AuthFormProps) {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const labels = content[mode];

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setIsSubmitting(true);
    const formData = new FormData(event.currentTarget);
    try {
      const response = await fetch(`/api/auth/${mode}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: formData.get("email"), password: formData.get("password") }),
      });
      if (!response.ok) {
        setError(await responseError(response));
        return;
      }
      router.replace("/dashboard");
      router.refresh();
    } catch {
      setError("Не удалось связаться с сервисом авторизации. Попробуйте ещё раз.");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <main className="auth-page">
      <section className="auth-card" aria-labelledby="auth-title">
        <Link className="brand" href="/">Uptime</Link>
        <h1 id="auth-title">{labels.title}</h1>
        <p className="auth-description">Используйте email и пароль не короче 8 символов.</p>
        <form className="auth-form" onSubmit={handleSubmit}>
          <label htmlFor="email">Email</label>
          <input id="email" name="email" type="email" autoComplete="email" required />
          <label htmlFor="password">Пароль</label>
          <input id="password" name="password" type="password" autoComplete={mode === "login" ? "current-password" : "new-password"} minLength={8} required />
          {error ? <p className="form-error" role="alert" aria-live="polite">{error}</p> : null}
          <button type="submit" disabled={isSubmitting}>{isSubmitting ? "Подождите…" : labels.submit}</button>
        </form>
        <p className="auth-switch">{labels.prompt} <Link href={labels.linkHref}>{labels.linkLabel}</Link></p>
      </section>
    </main>
  );
}

async function responseError(response: Response) {
  try {
    const body = (await response.json()) as ApiError;
    if (typeof body.error === "string") return body.error;
  } catch {
    // Use a safe fallback for malformed error responses.
  }
  return "Не удалось выполнить запрос. Попробуйте ещё раз.";
}
