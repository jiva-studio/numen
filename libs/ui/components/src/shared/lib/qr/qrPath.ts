/** The dark squares of a QR code as one SVG path, a row of runs at a time, `quiet` squares in from the edge. */
export function getQrPath(modules: readonly (readonly boolean[])[], quiet: number): string {
  const parts: string[] = []
  modules.forEach((row, y) => {
    let start = -1
    row.forEach((isDark, x) => {
      if (isDark && start < 0) start = x
      if ((!isDark || x === row.length - 1) && start >= 0) {
        const end = isDark ? x + 1 : x
        parts.push(`M${start + quiet} ${y + quiet}h${end - start}v1h${start - end}z`)
        start = -1
      }
    })
  })
  return parts.join('')
}
