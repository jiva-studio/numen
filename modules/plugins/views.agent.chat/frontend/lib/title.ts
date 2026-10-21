/**
 * Title formatting for agent tabs.
 */
export const createTabTitle = (question: string, most = 24): string => {
  const line =
    question
      .split('\n')
      .find((one) => one.trim() !== '')
      ?.trim() ?? ''
  if (line.length <= most) return line
  const prefix = line.slice(0, most)
  const space = prefix.lastIndexOf(' ')
  return `${(space > most / 3 ? prefix.slice(0, space) : prefix).trimEnd()}…`
}
