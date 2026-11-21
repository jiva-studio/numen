/** How far each row of a list is moved aside while the row at `from` is held over the place `to`. */
export function getRowShifts(
  count: number,
  from: number,
  to: number,
  pitch: number,
): readonly number[] {
  return Array.from({ length: count }, (_, at) => {
    if (at === from) return 0
    if (from < to && at > from && at <= to) return -pitch
    if (from > to && at >= to && at < from) return pitch
    return 0
  })
}
