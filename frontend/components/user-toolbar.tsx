"use client";

import Link from "next/link";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";

export type UserProfile = {
  id: string;
  email: string;
  name: string;
  avatar_url: string;
};

export function isUserProfile(value: unknown): value is UserProfile {
  if (!value || typeof value !== "object") return false;
  const profile = value as Record<string, unknown>;
  return typeof profile.id === "string" &&
    typeof profile.email === "string" &&
    typeof profile.name === "string" &&
    typeof profile.avatar_url === "string" &&
    (profile.avatar_url === "" || /^\/api\/v1\/avatars\/[0-9a-f-]{36}\.(png|jpg)$/.test(profile.avatar_url));
}

export function useUserProfile() {
  const router = useRouter();
  const hasRefreshed = useRef(false);
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [isLoggingOut, setIsLoggingOut] = useState(false);

  useEffect(() => {
    if (hasRefreshed.current) return;
    hasRefreshed.current = true;
    async function refreshSession() {
      try {
        const response = await fetch("/api/auth/refresh", { method: "POST" });
        if (!response.ok) {
          router.replace("/login");
          return;
        }
        const profileResponse = await fetch("/api/auth/profile");
        if (!profileResponse.ok) {
          router.replace("/login");
          return;
        }
        const value: unknown = await profileResponse.json();
        if (!isUserProfile(value)) {
          router.replace("/login");
          return;
        }
        setProfile(value);
      } catch {
        router.replace("/login");
      }
    }
    void refreshSession();
  }, [router]);

  async function logout() {
    setIsLoggingOut(true);
    try {
      await fetch("/api/auth/logout", { method: "POST" });
    } finally {
      router.replace("/login");
      router.refresh();
    }
  }

  return { profile, setProfile, isLoggingOut, logout };
}

type Session = ReturnType<typeof useUserProfile>;

export function UserToolbar({ session }: { session: Session }) {
  const displayName = session.profile?.name || session.profile?.email || "Загружаем профиль…";

  return (
    <header className="dashboard-header">
      <Link className="brand" href="/dashboard">Uptime</Link>
      <nav className="user-toolbar-nav" aria-label="Меню пользователя">
        <details className="profile-menu">
          <summary aria-label="Открыть меню пользователя">
            <span className="profile-avatar" aria-hidden="true">{session.profile?.avatar_url ? <Image unoptimized width={40} height={40} src={session.profile.avatar_url} alt="" /> : displayName.slice(0, 1).toUpperCase()}</span>
            <span className="profile-menu-label">
              <strong>{displayName}</strong>
              <small>{session.profile?.email || "Проверяем сессию"}</small>
            </span>
            <span aria-hidden="true">⌄</span>
          </summary>
          {session.profile && (
            <div className="profile-menu-popover">
              <Link href="/profile">Профиль</Link>
            </div>
          )}
        </details>
        {session.profile && (
          <button className="logout-button" type="button" onClick={session.logout} disabled={session.isLoggingOut}>
            {session.isLoggingOut ? "Выходим…" : "Выйти"}
          </button>
        )}
      </nav>
    </header>
  );
}
