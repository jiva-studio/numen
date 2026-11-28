/** How long a link still works, as "10 min" or "2 h". */
export const getTimeLeftText = (expiresAt: string, now: number): string => {
  const minutes = Math.max(0, Math.ceil((new Date(expiresAt).getTime() - now) / 60_000))
  return minutes < 60 ? `${minutes} min` : `${Math.ceil(minutes / 60)} h`
}
