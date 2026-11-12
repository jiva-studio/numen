/**
 * Navigation and submenu interaction within a menu.
 */
import {
  computed,
  getCurrentScope,
  onScopeDispose,
  ref,
  type ComponentPublicInstance,
  type ComputedRef,
  type Ref,
  type ShallowRef,
} from 'vue'
import type { Position } from '@/shared/lib/math/geometry'
import {
  getInitialIndex,
  groupItems,
  type GroupedItem,
  type MenuItem,
  type MenuTrigger,
} from '../items/item'
import { createSubmenuAim, type MenuBox } from './aim'
import { isMenuRowHandle, useMenuKeys, type MenuRowHandle } from './keys'
import { isInside } from './outside'

interface ActiveSubmenu {
  readonly id: string
  readonly items: readonly MenuItem[]
  readonly at: Position
  readonly from: HTMLElement | null
}

interface MenuSubmenuState {
  readonly active: Ref<ActiveSubmenu | null>
  readonly trigger: Ref<MenuTrigger>
  readonly onRegisterRow: (id: string, row: Element | ComponentPublicInstance | null) => void
  readonly onRowHover: (item: MenuItem, event?: MouseEvent) => void
  readonly onMenuLeave: (event?: MouseEvent) => void
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
  readonly getSubmenuBox?: () => MenuBox | null
  readonly getSubmenuElement?: () => HTMLElement | null
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
  readonly pointTo: (pos: number) => void
  readonly onKey: (event: KeyboardEvent) => void
  readonly onResetNavigation: () => void
}

function useSubmenuActions(options: MenuNavigationOptions) {
  const active = ref<ActiveSubmenu | null>(null)
  const trigger = ref<MenuTrigger>('pointer')
  const rowElements = new Map<string, MenuRowHandle | null>()

  const onOpenSubmenu = (item: MenuItem, how: MenuTrigger): void => {
    // Stryker disable next-line ConditionalExpression,LogicalOperator: callers check children before opening and TypeScript narrows item.children
    if (item.isDisabled || !item.children || item.children.length === 0) return
    const rowInst = rowElements.get(item.id)
    const el = rowInst?.getElement() ?? null
    const position = options.getSubmenuPosition(el) ?? options.getFallbackPosition()
    active.value = {
      id: item.id,
      items: item.children,
      at: position,
      from: el,
    }
    trigger.value = how
    aim.onSubmenuOpened()
  }

  const onCloseSubmenu = (): void => {
    aim.clear()
    active.value = null
  }

  const aim = createSubmenuAim({
    clock: options.clock,
    getActiveId: () => active.value?.id ?? null,
    getSubmenuBox: () => options.getSubmenuBox?.() ?? null,
    isInsideSubmenu: (event) => isInside(options.getSubmenuElement?.() ?? null, event),
    isInsideMenu: (event) => isInside(options.menuRef.value, event),
    onOpen: (item) => onOpenSubmenu(item, 'pointer'),
    onClose: onCloseSubmenu,
  })

  if (getCurrentScope()) {
    onScopeDispose(() => aim.clear())
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
    aim,
    onOpenSubmenu,
    onCloseSubmenu,
    onSubmenuChoose,
    onChoose,
  }
}

export function useMenuNavigation(options: MenuNavigationOptions): MenuNavigationState {
  const submenuActions = useSubmenuActions(options)
  const { onOpenSubmenu, onCloseSubmenu, onChoose } = submenuActions

  const onRegisterRow = (id: string, row: Element | ComponentPublicInstance | null): void => {
    const handle = isMenuRowHandle(row) ? row : null
    holdRow(id, handle)
    submenuActions.rowElements.set(id, handle)
  }

  const visibleItems = computed(() => options.items())

  const rows = computed(() => groupItems(visibleItems.value))

  const {
    here,
    holdRow,
    goTo,
    pointTo,
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
    onRowHover: submenuActions.aim.onRowHover,
    onMenuLeave: submenuActions.aim.onMenuLeave,
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
    pointTo,
    onResetNavigation,
  }
}
