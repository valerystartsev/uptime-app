"use client";

import { FormEvent, useEffect, useState } from "react";

import { UserToolbar, useUserProfile } from "@/components/user-toolbar";
import { IntervalUnit, isMonitor, isMonitorURL, Monitor } from "@/lib/monitors/types";

const unitLabels: Record<IntervalUnit, [string, string, string]> = {
  seconds: ["секунду", "секунды", "секунд"],
  minutes: ["минуту", "минуты", "минут"],
  hours: ["час", "часа", "часов"],
};

function intervalLabel(value: number, unit: IntervalUnit) {
  const lastTwo = value % 100;
  const last = value % 10;
  const form = lastTwo >= 11 && lastTwo <= 14 ? 2 : last === 1 ? 0 : last >= 2 && last <= 4 ? 1 : 2;
  return `Каждые ${value} ${unitLabels[unit][form]}`;
}

export function Dashboard() {
  const session = useUserProfile();

  const [monitors, setMonitors] = useState<Monitor[] | null>(null);
  const [loadError, setLoadError] = useState("");
  const [reloadKey, setReloadKey] = useState(0);
  const [formOpen, setFormOpen] = useState(false);
  const [url, setURL] = useState("");
  const [intervalValue, setIntervalValue] = useState("5");
  const [intervalUnit, setIntervalUnit] = useState<IntervalUnit>("seconds");
  const [formError, setFormError] = useState("");
  const [saving, setSaving] = useState(false);

  const profileID = session.profile?.id;
  useEffect(() => {
    if (!profileID) return;
    const controller = new AbortController();
    async function load() {
      setLoadError("");
      try {
        const response = await fetch("/api/monitors", { signal: controller.signal });
        if (!response.ok) throw new Error("Не удалось загрузить сайты");
        const value: unknown = await response.json();
        if (!Array.isArray(value) || !value.every(isMonitor)) throw new Error("Некорректный ответ сервиса");
        if (!controller.signal.aborted) setMonitors(value);
      } catch {
        if (!controller.signal.aborted) setLoadError("Не удалось загрузить сайты. Попробуйте ещё раз.");
      }
    }
    void load();
    return () => controller.abort();
  }, [profileID, reloadKey]);

  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalizedURL = url.trim();
    const number = Number(intervalValue);
    if (!isMonitorURL(normalizedURL) || !Number.isSafeInteger(number) || number <= 0) {
      setFormError("Укажите URL сайта и положительное целое число для интервала.");
      return;
    }
    setSaving(true);
    setFormError("");
    try {
      const response = await fetch("/api/monitors", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url: normalizedURL, interval_value: number, interval_unit: intervalUnit }),
      });
      const value: unknown = await response.json();
      if (!response.ok) {
        const message = value && typeof value === "object" && "error" in value && typeof value.error === "string"
          ? value.error : "Не удалось создать сайт";
        throw new Error(message);
      }
      if (!isMonitor(value)) throw new Error("Сервис вернул некорректный ответ");
      setMonitors((current) => [value, ...(current ?? [])]);
      setLoadError("");
      setFormOpen(false);
      setURL("");
      setIntervalValue("5");
      setIntervalUnit("seconds");
    } catch (error) {
      setFormError(error instanceof Error ? error.message : "Не удалось создать сайт");
    } finally {
      setSaving(false);
    }
  }

  return (
    <main className="dashboard-page">
      <UserToolbar session={session} />
      <section className="monitors-section" aria-labelledby="monitors-title">
        <div className="monitors-heading">
          <div>
            <p className="eyebrow">Мониторинг</p>
            <h1 id="monitors-title">Мои сайты</h1>
            <p>Добавьте сайт и задайте частоту будущих проверок.</p>
          </div>
          <button type="button" onClick={() => { setFormOpen(true); setFormError(""); }} disabled={!session.profile}>
            Добавить сайт
          </button>
        </div>

        {formOpen && (
          <form className="monitor-form" onSubmit={create}>
            <h2>Новый сайт</h2>
            <label htmlFor="monitor-url">URL сайта</label>
            <input id="monitor-url" type="url" value={url} onChange={(event) => setURL(event.target.value)}
              placeholder="https://example.com" required maxLength={2048} />
            <div className="monitor-interval-fields">
              <div>
                <label htmlFor="monitor-interval">Частота опроса</label>
                <input id="monitor-interval" type="number" min="1" step="1" required value={intervalValue}
                  onChange={(event) => setIntervalValue(event.target.value)} />
              </div>
              <div>
                <label htmlFor="monitor-unit">Единица измерения</label>
                <select id="monitor-unit" value={intervalUnit}
                  onChange={(event) => setIntervalUnit(event.target.value as IntervalUnit)}>
                  <option value="seconds">Секунды</option>
                  <option value="minutes">Минуты</option>
                  <option value="hours">Часы</option>
                </select>
              </div>
            </div>
            {formError && <p className="form-error" role="alert">{formError}</p>}
            <div className="monitor-form-actions">
              <button type="submit" disabled={saving}>{saving ? "Создаём…" : "Создать"}</button>
              <button className="button-secondary" type="button" disabled={saving} onClick={() => setFormOpen(false)}>Отмена</button>
            </div>
          </form>
        )}

        {loadError && <div className="monitor-message" role="alert">
          <p>{loadError}</p>
          <button type="button" onClick={() => setReloadKey((current) => current + 1)}>Повторить</button>
        </div>}
        {!loadError && monitors === null && <p className="monitor-message">Загружаем сайты…</p>}
        {!loadError && monitors?.length === 0 && <p className="monitor-message">Сайтов пока нет. Добавьте первый сайт.</p>}
        {monitors && monitors.length > 0 && (
          <ul className="monitor-list">
            {monitors.map((monitor) => (
              <li className="monitor-card" key={monitor.id}>
                <a href={monitor.url} target="_blank" rel="noopener noreferrer">{monitor.url}</a>
                <p>{intervalLabel(monitor.interval_value, monitor.interval_unit)}</p>
              </li>
            ))}
          </ul>
        )}
      </section>
    </main>
  );
}
