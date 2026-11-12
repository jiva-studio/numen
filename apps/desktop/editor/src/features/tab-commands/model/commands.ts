/**
 * The commands of the window over a tab: detach it, unlink it, move it to another window and link
 * it with another tab. Moving and linking offer one item per window and per tab.
 */
import type { CommandItem } from '@numen/plugin-loader'
import { type Command, type FeatureCommands } from '@/entities/command'
import { COMMAND, type CommandInvocation, type CommandTarget } from '@/shared/model/command'
import {
  getAllTabs,
  getEligibleTabs,
  getWindowName,
  type LinkableTab,
  type TabId,
  type TabLinkGroup,
  type TabWindow,
} from '@/entities/tab'
import { TAB_WORDS } from '../words'

/** The tabs of every window and the links between them, as the commands read and change them. */
interface TabLinks {
  readonly tabs: readonly LinkableTab[]
  readonly windows: readonly TabWindow[]
  linkTabs(sourceTabId: TabId, targetTabId: TabId): void
  unlinkTab(tabId: TabId): void
  getGroupOf(this: void, tabId: TabId): TabLinkGroup | null
}

interface TabEntry {
  readonly id: TabId
  readonly title: string
}

export interface TabCommandsDeps {
  readonly tabLinks: TabLinks
  /** The tabs of this window. */
  getTabs(): readonly TabEntry[]
  getKind(tab: TabId): string | undefined
  /** Whether a tab of this window can be linked. */
  isLinkable(tab: TabId): boolean
  readonly windowId?: string
  /** Whether a tab is asking; an asking tab is not moved or detached. */
  isAsking(tab: TabId): boolean
  detach(tab: TabId): void
  move(tab: TabId, windowId: string): void
}

const createRows = (deps: TabCommandsDeps): readonly Command[] => {
  const { tabLinks } = deps
  const getKind = (tab: TabId): string | undefined =>
    deps.getKind(tab) ?? tabLinks.tabs.find((one) => one.id === tab)?.kind
  const isLinkable = (tab: TabId): boolean =>
    deps.getTabs().some((one) => one.id === tab)
      ? deps.isLinkable(tab)
      : tabLinks.tabs.some((one) => one.id === tab)
  const getWindows = () => tabLinks.windows.filter((one) => one.id !== deps.windowId)
  const isMovable = (target: CommandTarget) => target.tab !== '' && !deps.isAsking(target.tab)

  const getLinkItems = (target: CommandTarget): readonly CommandItem[] =>
    getEligibleTabs({
      from: target.tab,
      tabs: getAllTabs(deps.getTabs(), tabLinks.tabs),
      getKind,
      isLinkable,
      getGroupOf: tabLinks.getGroupOf,
    }).map((tab) => ({ id: tab.id, title: tab.title || tab.kind }))

  return [
    {
      id: COMMAND.detachTab,
      text: TAB_WORDS.detach,
      group: 'window',
      isAvailable: isMovable,
    },
    {
      id: COMMAND.unlinkTab,
      text: TAB_WORDS.unlink,
      group: 'window',
      isAvailable: (target) => target.tab !== '' && tabLinks.getGroupOf(target.tab) !== null,
    },
    {
      id: COMMAND.moveTabToWindow,
      text: TAB_WORDS.moveToWindow,
      group: 'window',
      input: 'items',
      isAvailable: (target) => isMovable(target) && getWindows().length > 0,
      listItems: () => getWindows().map((one) => ({ id: one.id, title: getWindowName(one) })),
    },
    {
      id: COMMAND.linkTab,
      text: TAB_WORDS.link,
      group: 'window',
      input: 'items',
      isAvailable: (target) => target.tab !== '' && isLinkable(target.tab),
      listItems: getLinkItems,
    },
  ]
}

const requireTab =
  (run: (invocation: CommandInvocation) => void) =>
  (invocation: CommandInvocation): void => {
    if (invocation.target.tab !== '') run(invocation)
  }

const withItem = (run: (tab: TabId, itemId: string) => void) =>
  requireTab((invocation) => {
    if (invocation.name !== '') run(invocation.target.tab, invocation.name)
  })

/** The commands over a tab. */
export const createTabCommands = (deps: TabCommandsDeps): FeatureCommands => ({
  commands: createRows(deps),
  handlers: {
    [COMMAND.detachTab]: requireTab((invocation) => deps.detach(invocation.target.tab)),
    [COMMAND.unlinkTab]: requireTab((invocation) => deps.tabLinks.unlinkTab(invocation.target.tab)),
    [COMMAND.moveTabToWindow]: withItem((tab, itemId) => deps.move(tab, itemId)),
    [COMMAND.linkTab]: withItem((tab, itemId) => deps.tabLinks.linkTabs(tab, itemId)),
  },
})
