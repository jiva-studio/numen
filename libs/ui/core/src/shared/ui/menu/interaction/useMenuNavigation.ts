/**
 * Navigation and submenu interaction within a menu.
 */
import { computed, ref, type ComputedRef, type Ref, type ShallowRef } from 'vue'
import type { Position } from '@/shared/lib/math/geometry'
import {
  getInitialIndex,
  groupItems,
  type GroupedItem,
  type MenuItem,
  type MenuTrigger,
} from '../items/item'
import { useMenuKeys } from './keys'

interface ActiveSubmenu {
  readonly id: string
  readonly items: readonly MenuItem[]
  readonly at: Position
  readonly from: HTMLElement | null
}

interface MenuSubmenuState {
  readonly active: Ref<ActiveSubmenu | null>
  readonly trigger: Ref<MenuTrigger>
  readonly onRegisterRow: (id: string, row: unknown) => void
  readonly onRowHover: (item: MenuItem) => void
  readonly onChoose: (item: MenuItem) => void
  readonly onSubmenuChoose: (id: string) => void
  readonly onClose: () => void
}

export interface MenuNavigationOptions {
  readonly items: () => readonly MenuItem[]
  readonly trigger: () => MenuTrigger
  readonly current: () => string | null
  readonly clock: () => number
  readonly menuRef: Readonly<ShallowRef<HTMLElement | null>>
  readonly getFallbackPosition: () => Position
  readonly getSubmenuPosition: (element: HTMLElement | null) => Position | null
  readonly onEmitChoose: (id: string) => void
  readonly onEmitDismiss: () => void
  readonly hasFrom: () => boolean
}

export interface MenuNavigationState {
  readonly visibleItems: ComputedRef<readonly MenuItem[]>
  readonly rows: ComputedRef<readonly GroupedItem[]>
  readonly here: Ref<number>
  readonly submenu: MenuSubmenuState
  readonly focusInitial: () => void
  readonly onKey: (event: KeyboardEvent) => void
  readonly onResetNavigation: () => void
}

/** The submenu that is open, and what opening, closing and choosing do with it. */
function useSubmenuActions(options: MenuNavigationOptions) {
  const active = ref<ActiveSubmenu | null>(null)
  const trigger = ref<MenuTrigger>('pointer')
  const rowElements = new Map<string, { getElement?: () => HTMLElement | null }>()

  const onOpenSubmenu = (item: MenuItem, how: MenuTrigger = 'pointer'): void => {
    if (!item.children || item.children.length === 0) return
    const rowInst = rowElements.get(item.id)
    const el = rowInst?.getElement?.() ?? null
    const position = options.getSubmenuPosition(el) ?? options.getFallbackPosition()
    active.value = {
      id: item.id,
      items: item.children,
      at: position,
      from: el,
    }
    trigger.value = how
  }

  const onCloseSubmenu = (): void => {
    active.value = null
  }

  const onRowHover = (item: MenuItem): void => {
    if (item.children && item.children.length > 0) {
      onOpenSubmenu(item, 'pointer')
    } else if (active.value && active.value.id !== item.id) {
      onCloseSubmenu()
    }
  }

  const onSubmenuChoose = (id: string): void => {
    onCloseSubmenu()
    options.onEmitChoose(id)
    options.onEmitDismiss()
  }

  const onChoose = (item: MenuItem): void => {
    if (item.isDisabled) return
    if (item.children && item.children.length > 0) {
      onOpenSubmenu(item, 'pointer')
      return
    }
    options.onEmitChoose(item.id)
    options.onEmitDismiss()
  }

  return {
    active,
    trigger,
    rowElements,
    onOpenSubmenu,
    onCloseSubmenu,
    onRowHover,
    onSubmenuChoose,
    onChoose,
  }
}

export function useMenuNavigation(options: MenuNavigationOptions): MenuNavigationState {
  const submenuActions = useSubmenuActions(options)
  const { onOpenSubmenu, onCloseSubmenu, onChoose } = submenuActions

  const onRegisterRow = (id: string, row: unknown): void => {
    holdRow(id, row)
    submenuActions.rowElements.set(id, row as { getElement?: () => HTMLElement | null })
  }

  const visibleItems = computed(() => options.items())

  const rows = computed(() => groupItems(visibleItems.value))

  const {
    here,
    holdRow,
    goTo,
    onKey: onRowKey,
  } = useMenuKeys(() => visibleItems.value, options.menuRef, options.clock, {
    onOpenSubmenu: (item) => onOpenSubmenu(item, 'keyboard'),
    onDismissSubmenu: options.hasFrom() ? options.onEmitDismiss : undefined,
  })

  const onResetNavigation = (): void => {
    here.value = -1
    onCloseSubmenu()
  }

  const submenu: MenuSubmenuState = {
    active: submenuActions.active,
    trigger: submenuActions.trigger,
    onRegisterRow,
    onRowHover: submenuActions.onRowHover,
    onChoose,
    onSubmenuChoose: submenuActions.onSubmenuChoose,
    onClose: onCloseSubmenu,
  }

  return {
    visibleItems,
    rows,
    here,
    submenu,
    focusInitial: () =>
      goTo(getInitialIndex(options.trigger(), options.items(), options.current())),
    onKey: onRowKey,
    onResetNavigation,
  }
}
