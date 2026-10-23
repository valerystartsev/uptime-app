export type IntervalUnit = "seconds" | "minutes" | "hours";

export type Monitor = {
  id: string;
  url: string;
  interval_value: number;
  interval_unit: IntervalUnit;
  created_at: string;
};

export function isIntervalUnit(value: unknown): value is IntervalUnit {
  return value === "seconds" || value === "minutes" || value === "hours";
}

export function isMonitor(value: unknown): value is Monitor {
  if (!value || typeof value !== "object" || Array.isArray(value)) return false;
  const monitor = value as Record<string, unknown>;
  return typeof monitor.id === "string" && /^[0-9a-f-]{36}$/.test(monitor.id) &&
    typeof monitor.url === "string" && isMonitorURL(monitor.url) &&
    typeof monitor.interval_value === "number" && Number.isSafeInteger(monitor.interval_value) &&
    monitor.interval_value > 0 && isIntervalUnit(monitor.interval_unit) &&
    typeof monitor.created_at === "string" && !Number.isNaN(Date.parse(monitor.created_at));
}

export function isMonitorURL(value: string): boolean {
  if (!value || value.length > 2048 || value !== value.trim()) return false;
  try {
    const url = new URL(value);
    return (url.protocol === "http:" || url.protocol === "https:") &&
      Boolean(url.hostname) && !url.username && !url.password && !url.hash;
  } catch {
    return false;
  }
}
