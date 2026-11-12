/** The list of vaults the title opens: its rows, its position and the effect of choosing a row. */
import { computed, onBeforeUnmount, ref, watch, type ComputedRef, type Ref } from 'vue'
import { measureRect, type ListItem } from '@numen/ui'
import type { VaultsState } from '@numen/plugin-sdk'
import { TITLE_BAR_WORDS } from '../lib/words'

/** The id of the last row of the list; its NUL character keeps it apart from every vault id. */
const CREATE_VAULT_ROW = '\u0000create-vault'

export interface VaultListSources {
  readonly trigger: Readonly<Ref<HTMLElement | null>>
  readonly getVaults: () => VaultsState
  readonly canCreate: () => boolean
  readonly onCreate: () => void
  readonly onSwitch: (id: string) => void
}

/** What the popup that draws the list is given. */
interface VaultListPopup {
  readonly items: readonly ListItem[]
  readonly modelValue: string | null
  readonly at: { readonly x: number; readonly y: number }
  readonly asking: number
}

export interface VaultList {
  readonly hasVaults: ComputedRef<boolean>
  readonly isListOpen: ComputedRef<boolean>
  readonly popup: ComputedRef<VaultListPopup>
  readonly toggle: () => void
  readonly select: (id: string) => void
  readonly close: () => void
}

export function useVaultList(sources: VaultListSources): VaultList {
  const { trigger, getVaults, canCreate, onCreate, onSwitch } = sources
  const isOpen = ref(false)
  const anchor = ref({ x: 0, y: 0 })
  const triggerWidth = ref(0)
  const hasVaults = computed(() => getVaults().current !== null && getVaults().vaults.length > 0)
  const isListOpen = computed(() => isOpen.value && hasVaults.value)
  const popup = computed<VaultListPopup>(() => ({
    items: [
      ...getVaults().vaults.map(({ id, name }) => ({ id, text: name })),
      ...(canCreate()
        ? [{ id: CREATE_VAULT_ROW, text: TITLE_BAR_WORDS.createVault, hasSeparator: true }]
        : []),
    ],
    modelValue: getVaults().current?.id ?? null,
    at: anchor.value,
    asking: triggerWidth.value,
  }))

  function onPressTitle(event: Event): void {
    if (event.target instanceof Node && trigger.value?.contains(event.target)) {
      event.stopImmediatePropagation()
    }
  }

  // Registered before the list is drawn, so it runs before the list's own press outside: a press of
  // the title is the title's to answer, which closes the list through the click that follows.
  watch(
    isListOpen,
    (open) => {
      if (open) globalThis.addEventListener('pointerdown', onPressTitle, true)
      else globalThis.removeEventListener('pointerdown', onPressTitle, true)
    },
    { flush: 'sync' },
  )

  onBeforeUnmount(() => globalThis.removeEventListener('pointerdown', onPressTitle, true))

  function toggle(): void {
    if (isOpen.value) {
      isOpen.value = false
      return
    }
    const box = measureRect(trigger.value)
    anchor.value = { x: box?.left ?? 0, y: box?.bottom ?? 0 }
    triggerWidth.value = box?.width ?? 0
    isOpen.value = true
  }

  function select(id: string): void {
    isOpen.value = false
    if (id === CREATE_VAULT_ROW) onCreate()
    else if (id !== (getVaults().current?.id ?? null)) onSwitch(id)
  }

  function close(): void {
    isOpen.value = false
  }

  return { hasVaults, isListOpen, popup, toggle, select, close }
}
