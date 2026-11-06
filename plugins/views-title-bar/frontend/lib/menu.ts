import type { MenuItem } from '@numen/ui'
import type { SlotItem } from '@numen/plugin-sdk'

/** The rows of a menu the bar opens, and the command and the icon of each row by its id. */
interface BarMenu {
  readonly items: readonly MenuItem[]
  readonly commands: ReadonlyMap<string, string>
  readonly icons: ReadonlyMap<string, unknown>
}

interface MenuParts {
  readonly commands: Map<string, string>
  readonly icons: Map<string, unknown>
}

function createItems(entries: readonly SlotItem[], path: string, parts: MenuParts): MenuItem[] {
  const items: MenuItem[] = []
  let group = 0
  entries.forEach((entry, index) => {
    const id = `${path}${index}`
    if (entry.kind === 'separator') group += 1
    if (entry.kind === 'command') {
      parts.commands.set(id, entry.command)
      if (entry.icon !== undefined) parts.icons.set(id, entry.icon)
      items.push({ id, text: entry.title, group: String(group) })
    }
    if (entry.kind === 'submenu') {
      if (entry.icon !== undefined) parts.icons.set(id, entry.icon)
      const children = createItems(entry.entries, `${id}.`, parts)
      items.push({ id, text: entry.title, group: String(group), children })
    }
  })
  return items
}

/**
 * The menu of the entries given: a command is a row, a submenu a row that opens its entries, and a
 * separator starts a group. A view is not drawn in a menu.
 */
export function createBarMenu(entries: readonly SlotItem[]): BarMenu {
  const parts: MenuParts = { commands: new Map(), icons: new Map() }
  return { items: createItems(entries, '', parts), ...parts }
}
