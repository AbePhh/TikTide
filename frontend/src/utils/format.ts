export function formatCount(value: number): string {
  if (value >= 100000000) {
    return `${(value / 100000000).toFixed(1).replace(/\.0$/, "")}亿`;
  }
  if (value >= 10000) {
    return `${(value / 10000).toFixed(1).replace(/\.0$/, "")}万`;
  }
  return String(value);
}

export function formatRelativeTime(iso: string): string {
  const time = new Date(iso).getTime();
  if (Number.isNaN(time)) {
    return iso;
  }

  const diff = Date.now() - time;
  const minute = 60 * 1000;
  const hour = 60 * minute;
  const day = 24 * hour;

  if (diff < minute) {
    return "刚刚";
  }
  if (diff < hour) {
    return `${Math.max(1, Math.floor(diff / minute))}分钟前`;
  }
  if (diff < day) {
    return `${Math.max(1, Math.floor(diff / hour))}小时前`;
  }
  if (diff < day * 7) {
    return `${Math.max(1, Math.floor(diff / day))}天前`;
  }

  return new Date(iso).toLocaleDateString("zh-CN", {
    month: "2-digit",
    day: "2-digit"
  });
}

export function buildAvatarFallback(name: string): string {
  const value = name.trim();
  if (!value) {
    return "T";
  }
  return value.slice(0, 1).toUpperCase();
}

