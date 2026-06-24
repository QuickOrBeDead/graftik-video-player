export function formatTime(seconds?: number | undefined) {
  if (seconds === undefined || isNaN(seconds) || seconds < 0) {
    return '0:00'
  }

  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = Math.floor(seconds % 60)
  return `${h > 0 ? h + ':' : ''}${m < 10 ? '0' + m : m}:${s < 10 ? '0' + s : s}`
}