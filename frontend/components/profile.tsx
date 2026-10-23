"use client";

import Link from "next/link";
import Image from "next/image";
import { useEffect, useState, type ChangeEvent, type FormEvent } from "react";

import { UserToolbar, isUserProfile, useUserProfile } from "@/components/user-toolbar";

export function Profile() {
  const session = useUserProfile();
  const [name, setName] = useState("");
  const [isEditing, setIsEditing] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState("");
  const [avatar, setAvatar] = useState<File | null>(null);
  const [preview, setPreview] = useState("");

  useEffect(() => () => { if (preview) URL.revokeObjectURL(preview); }, [preview]);

  function selectAvatar(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    if (!["image/png", "image/jpeg"].includes(file.type) || file.size > 5 * 1024 * 1024) {
      setError("Выберите JPEG или PNG размером до 5 МБ.");
      event.target.value = "";
      return;
    }
    setError("");
    setAvatar(file);
    setPreview(URL.createObjectURL(file));
  }

  function startEditing() {
    setName(session.profile?.name ?? "");
    setAvatar(null);
    setPreview("");
    setError("");
    setIsEditing(true);
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmedName = name.trim();
    if (!trimmedName || [...trimmedName].length > 100) {
      setError("Имя должно содержать от 1 до 100 символов.");
      return;
    }
    setIsSaving(true);
    setError("");
    try {
      const response = await fetch("/api/auth/profile", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: trimmedName }),
      });
      if (!response.ok) {
        setError(response.status === 400 ? "Проверьте имя и попробуйте снова." : "Не удалось сохранить профиль.");
        return;
      }
      const value: unknown = await response.json();
      if (!isUserProfile(value)) {
        setError("Не удалось сохранить профиль.");
        return;
      }
      let savedProfile = value;
      if (avatar) {
        const form = new FormData();
        form.set("avatar", avatar);
        const uploadResponse = await fetch("/api/auth/profile/avatar", { method: "POST", body: form });
        if (!uploadResponse.ok) {
          setError(uploadResponse.status === 400 ? "Выберите корректный JPEG или PNG размером до 5 МБ." : "Не удалось загрузить аватар.");
          session.setProfile(value);
          return;
        }
        const uploaded: unknown = await uploadResponse.json();
        if (!isUserProfile(uploaded)) {
          setError("Не удалось загрузить аватар.");
          return;
        }
        savedProfile = uploaded;
      }
      session.setProfile(savedProfile);
      setAvatar(null);
      setPreview("");
      setIsEditing(false);
    } catch {
      setError("Не удалось сохранить профиль.");
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <main className="dashboard-page">
      <UserToolbar session={session} />
      {session.profile && (
        <section className="profile-card" aria-labelledby="profile-title">
          <Link className="back-link" href="/dashboard">← На главную</Link>
          <h1 id="profile-title">Профиль</h1>
          <div className="profile-picture">
            {preview || session.profile.avatar_url ? <Image unoptimized width={96} height={96} src={preview || session.profile.avatar_url} alt="Аватар профиля" /> :
              <span aria-label="Аватар профиля">{(session.profile.name || session.profile.email).slice(0, 1).toUpperCase()}</span>}
          </div>
          {isEditing ? (
            <form onSubmit={save} className="profile-form">
              <label htmlFor="profile-avatar-upload">Загрузить аватар</label>
              <input id="profile-avatar-upload" type="file" accept="image/jpeg,image/png" onChange={selectAvatar} />
              <label htmlFor="profile-name">Имя</label>
              <input id="profile-name" name="name" autoComplete="name" maxLength={100} value={name}
                onChange={(event) => setName(event.target.value)} required />
              <p className="profile-hint">Email: {session.profile.email}</p>
              {error && <p className="form-error" role="alert">{error}</p>}
              <div className="profile-actions">
                <button type="submit" disabled={isSaving}>{isSaving ? "Сохраняем…" : "Сохранить"}</button>
                <button className="button-secondary" type="button" disabled={isSaving} onClick={() => {
                  setError("");
                  setAvatar(null);
                  setPreview("");
                  setIsEditing(false);
                }}>Отмена</button>
              </div>
            </form>
          ) : (
            <div className="profile-details">
              <div><span>Имя</span><strong>{session.profile.name || "Не указано"}</strong></div>
              <div><span>Email</span><strong>{session.profile.email}</strong></div>
              <button type="button" onClick={startEditing}>Редактировать</button>
            </div>
          )}
        </section>
      )}
    </main>
  );
}
