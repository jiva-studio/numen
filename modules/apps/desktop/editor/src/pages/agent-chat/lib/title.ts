/**
 * Title formatting for agent tabs.
 */
export const firstLine = (question: string, most = 24): string => {
  const line =
    question
      .split('\n')
      .find((one) => one.trim() !== '')
      ?.trim() ?? ''
  if (line.length <= most) return line
  const cut = line.slice(0, most)
  const space = cut.lastIndexOf(' ')
  return `${(space > most / 3 ? cut.slice(0, space) : cut).trimEnd()}…`
}
