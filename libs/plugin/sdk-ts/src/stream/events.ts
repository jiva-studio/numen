/** Splits the text of an event stream into its events as they complete. */
export const parseEvents = (
  text: string,
): { events: { name: string; data: string }[]; rest: string } => {
  const events: { name: string; data: string }[] = []
  const blocks = text.split('\n\n')
  const rest = blocks.pop() ?? ''
  for (const block of blocks) {
    let name = 'message'
    const data: string[] = []
    for (const line of block.split('\n')) {
      if (line.startsWith('event: ')) name = line.slice(7)
      else if (line.startsWith('data: ')) data.push(line.slice(6))
    }
    events.push({ name, data: data.join('\n') })
  }
  return { events, rest }
}
