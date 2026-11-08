/**
 * The commands of the window over a tab: detach it, unlink it, move it to another window and link
 * it with another tab. Moving and linking offer one item per window and per tab.
 */
import type { CommandItem } from '@numen/plugin-host'
import {
  registerCommand,
  type Command,
  type CommandInvocation,
  type CommandTarget,
} from '@/features/command-palette'
import {
  getAllTabs,
  getEligibleTabs,
  getWindowName,
  type TabId,
  type TabLinksState,
} from '@/features/tab-linking'
import { COMMAND } from '@/shared/commandIds'

export const TAB_COMMAND = {
  detach: COMMAND.detachTab,
  unlink: COMMAND.unlinkTab,
  moveToWindow: COMMAND.moveTabToWindow,
  link: COMMAND.linkTab,
} as const

const TAB_WORDS = {
  detach: 'Detach tab',
  unlink: 'Unlink tab',
  moveToWindow: 'Move to window',
  link: 'Link with tab',
}

export interface TabCommandsDeps {
  readonly tabLinks: TabLinksState
  /** The tabs of this window. */
  getTabs(): readonly { readonly id: TabId; readonly title: string }[]
  getKind(tab: TabId): string | undefined
  /** Whether a tab of this window can be linked. */
  isLinkable(tab: TabId): boolean
  readonly windowId?: string
  /** Whether a tab is asking; an asking tab is not moved or detached. */
  isAsking(tab: TabId): boolean
  detach(tab: TabId): void
  move(tab: TabId, windowId: string): void
}

export function createTabCommands(deps: TabCommandsDeps) {
  const getKind = (tab: TabId): string | undefined =>
    deps.getKind(tab) ?? deps.tabLinks.tabs.find((one) => one.id === tab)?.kind
  const isLinkable = (tab: TabId): boolean =>
    deps.getTabs().some((one) => one.id === tab)
      ? deps.isLinkable(tab)
      : deps.tabLinks.tabs.some((one) => one.id === tab)
  const getWindows = () => deps.tabLinks.windows.filter((one) => one.id !== deps.windowId)
  const isMovable = (target: CommandTarget) => target.tab !== '' && !deps.isAsking(target.tab)

  const getLinkItems = (target: CommandTarget): readonly CommandItem[] =>
    getEligibleTabs({
      from: target.tab,
      tabs: getAllTabs(deps.getTabs(), deps.tabLinks.tabs),
      getKind,
      isLinkable,
      getGroupOf: deps.tabLinks.getGroupOf,
    }).map((tab) => ({ id: tab.id, title: tab.title || tab.kind }))

  const list: readonly Command[] = [
    {
      id: TAB_COMMAND.detach,
      text: TAB_WORDS.detach,
      group: 'window',
      isAvailable: isMovable,
    },
    {
      id: TAB_COMMAND.unlink,
      text: TAB_WORDS.unlink,
      group: 'window',
      isAvailable: (target) => target.tab !== '' && deps.tabLinks.getGroupOf(target.tab) !== null,
    },
    {
      id: TAB_COMMAND.moveToWindow,
      text: TAB_WORDS.moveToWindow,
      group: 'window',
      input: 'items',
      isAvailable: (target) => isMovable(target) && getWindows().length > 0,
      listItems: () => getWindows().map((one) => ({ id: one.id, title: getWindowName(one) })),
    },
    {
      id: TAB_COMMAND.link,
      text: TAB_WORDS.link,
      group: 'window',
      input: 'items',
      isAvailable: (target) => target.tab !== '' && isLinkable(target.tab),
      listItems: getLinkItems,
    },
  ]

  /** Registers the runs of the commands; the result takes them back. */
  const register = (): (() => void) => {
    const requireTab =
      (run: (invocation: CommandInvocation) => void) => (invocation: CommandInvocation) => {
        if (invocation.tab !== '') run(invocation)
      }
    const withArgument = (run: (tab: TabId, argument: string) => void) =>
      requireTab((invocation) => {
        if (invocation.name !== '') run(invocation.tab, invocation.name)
      })
    const unregisterAll = [
      registerCommand(
        TAB_COMMAND.detach,
        requireTab((invocation) => deps.detach(invocation.tab)),
      ),
      registerCommand(
        TAB_COMMAND.unlink,
        requireTab((invocation) => deps.tabLinks.unlinkTab(invocation.tab)),
      ),
      registerCommand(
        TAB_COMMAND.moveToWindow,
        withArgument((tab, argument) => deps.move(tab, argument)),
      ),
      registerCommand(
        TAB_COMMAND.link,
        withArgument((tab, argument) => deps.tabLinks.linkTabs(tab, argument)),
      ),
    ]
    return () => {
      for (const unregister of unregisterAll) unregister()
    }
  }

  return { commands: list, register }
}

export type TabCommands = ReturnType<typeof createTabCommands>
