/** What is true of every workspace an edit produces. */
import { isBranch, panesOf, type Workspace, type WorkspaceNode } from '../model'

/** Anything that does not hold about a workspace, said in words. */
export function broken(workspace: Workspace): readonly string[] {
  const faults: string[] = []
  const seen = new Set<string>()

  const walk = (node: WorkspaceNode): void => {
    if (seen.has(node.id)) faults.push(`two nodes are called ${node.id}`)
    seen.add(node.id)

    if (!isBranch(node)) return

    if (node.children.length < 2) {
      faults.push(`branch ${node.id} divides its length between ${node.children.length}`)
    }
    if (node.sizes.length !== node.children.length) {
      faults.push(`branch ${node.id} has ${node.sizes.length} shares for ${node.children.length}`)
    }

    const total = node.sizes.reduce((sum, size) => sum + size, 0)
    if (Math.abs(total - 1) > 1e-9) faults.push(`shares of ${node.id} come to ${total}`)
    if (node.sizes.some((size) => !(size > 0))) faults.push(`branch ${node.id} gives a child nothing`)

    node.children.forEach(walk)
  }

  walk(workspace.root)

  const panes = panesOf(workspace.root)
  const bare = panes.filter((each) => each.tabs.length === 0)
  if (bare.length > 0 && panes.length > 1) {
    faults.push(`${bare.length} pane(s) hold nothing while others do`)
  }

  for (const each of panes) {
    if (each.active !== null && !each.tabs.includes(each.active)) {
      faults.push(`pane ${each.id} shows ${each.active}, which it does not hold`)
    }
    if (each.active === null && each.tabs.length > 0) {
      faults.push(`pane ${each.id} holds tabs and shows none`)
    }
  }

  if (!panes.some((each) => each.id === workspace.focus)) {
    faults.push(`the focus is on ${workspace.focus}, which is not a pane`)
  }

  const tabs = panes.flatMap((each) => each.tabs)
  if (new Set(tabs).size !== tabs.length) faults.push('a tab is open in two places')

  return faults
}
