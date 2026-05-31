const DATE_ONLY_PATTERN = /^(\d{4})-(\d{2})-(\d{2})$/

export function formatDateOnly(value: string, options: Intl.DateTimeFormatOptions): string {
  const match = DATE_ONLY_PATTERN.exec(value)
  if (!match) {
    return new Date(value).toLocaleDateString('en-ZA', options)
  }

  const [, year, month, day] = match
  const date = new Date(Date.UTC(Number(year), Number(month) - 1, Number(day)))
  return date.toLocaleDateString('en-ZA', { ...options, timeZone: 'UTC' })
}

export function formatShortDate(value: string): string {
  return formatDateOnly(value, {
    day: 'numeric',
    month: 'short',
  })
}
