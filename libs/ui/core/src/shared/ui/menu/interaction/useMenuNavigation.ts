/**
 * Navigation, search, and submenu interaction within a menu.
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
import { useMenuSearch } from '../items/search'

export interface ActiveSubmenu {
  readonly id: string
  readonly items: readonly MenuItem[]
  readonly at: Position
  readonly from: HTMLElement | null
}

export interface MenuSubmenuState {
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
  readonly isSearchable: () => boolean
  readonly canEnterCustom?: (() => boolean) | undefined
  readonly trigger: () => MenuTrigger
  readonly current: () => string | null
  readonly clock: () => number
  readonly searchRef: Readonly<ShallowRef<HTMLInputElement | null>>
  readonly menuRef: Readonly<ShallowRef<HTMLElement | null>>
  readonly getFallbackPosition: () => Position
  readonly getSubmenuPosition: (element: HTMLElement | null) => Position | null
  readonly onEmitChoose: (id: string) => void
  readonly onEmitDismiss: () => void
  readonly hasFrom: () => boolean
}

export interface MenuNavigationState {
  readonly query: Ref<string>
  readonly visibleItems: ComputedRef<readonly MenuItem[]>
  readonly rows: ComputedRef<readonly GroupedItem[]>
  readonly here: Ref<number>
  readonly submenu: MenuSubmenuState
  readonly focusInitial: () => void
  readonly onKey: (event: KeyboardEvent) => void
  readonly onResetNavigation: () => void
}

export function useMenuNavigation(options: MenuNavigationOptions): MenuNavigationState {
  const activeSubmenu = ref<ActiveSubmenu | null>(null)
  const submenuTrigger = ref<MenuTrigger>('pointer')
  const rowElements = new Map<string, { getElement?: () => HTMLElement | null }>()

  const onRegisterRow = (id: string, row: unknown): void => {
    holdRow(id, row)
    rowElements.set(id, row as { getElement?: () => HTMLElement | null })
  }

  const onOpenSubmenu = (item: MenuItem, trigger: MenuTrigger = 'pointer'): void => {
    if (!item.children || item.children.length === 0) return
    const rowInst = rowElements.get(item.id)
    const el = rowInst?.getElement?.() ?? null
    const position = options.getSubmenuPosition(el) ?? options.getFallbackPosition()
    activeSubmenu.value = {
      id: item.id,
      items: item.children,
      at: position,
      from: el,
    }
    submenuTrigger.value = trigger
  }

  const onCloseSubmenu = (): void => {
    activeSubmenu.value = null
  }

  const onRowHover = (item: MenuItem): void => {
    if (item.children && item.children.length > 0) {
      onOpenSubmenu(item, 'pointer')
    } else if (activeSubmenu.value && activeSubmenu.value.id !== item.id) {
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

  const {
    query,
    visibleItems,
    onKey: onSearchKey,
    focusInitial,
  } = useMenuSearch({
    isSearchable: options.isSearchable,
    canEnterCustom: options.canEnterCustom,
    items: options.items,
    searchRef: options.searchRef,
    menuRef: options.menuRef,
    onRowKey: (event) => onRowKey(event),
    onChoose,
    onResetHere: () => {
      here.value = -1
    },
    goToInitial: () => goTo(getInitialIndex(options.trigger(), options.items(), options.current())),
  })

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
    query.value = ''
    onCloseSubmenu()
  }

  const submenu: MenuSubmenuState = {
    active: activeSubmenu,
    trigger: submenuTrigger,
    onRegisterRow,
    onRowHover,
    onChoose,
    onSubmenuChoose,
    onClose: onCloseSubmenu,
  }

  return {
    query,
    visibleItems,
    rows,
    here,
    submenu,
    focusInitial,
    onKey: onSearchKey,
    onResetNavigation,
  }
}
