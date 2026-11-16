/** A length of time in milliseconds, written as a clock. */
export const clock = (ms: number): string => {
  const whole = Math.max(0, Math.floor(ms / 1000))
  const hours = Math.floor(whole / 3600)
  const minutes = Math.floor(whole / 60) % 60
  const seconds = `${whole % 60}`.padStart(2, '0')
  if (hours === 0) return `${minutes}:${seconds}`
  return `${hours}:${`${minutes}`.padStart(2, '0')}:${seconds}`
}
