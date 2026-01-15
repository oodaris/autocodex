export function formatTimestamp(value?: string | null): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

export function statusLabel(status: string): string {
  if (status === 'running') return 'Running'
  if (status === 'failed') return 'Failed'
  if (status === 'completed') return 'Completed'
  if (status === 'queued') return 'Queued'
  if (status === 'canceled') return 'Canceled'
  return status
}

export function formatBytes(value: number): string {
  if (value < 1024) return `${value} B`
  const units = ['KB', 'MB', 'GB', 'TB']
  let size = value
  let unitIndex = 0
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex += 1
  }
  return `${size.toFixed(1)} ${units[unitIndex]}`
}
