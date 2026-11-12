/**
 * What the palette draws: the groups of the current step, and the kind and icon of a row.
 */
import { computed, type Component } from 'vue'
import type { Vault } from '@/entities/vault'
import { getPaletteView } from '../view/view'
import type { PaletteSession } from './session'

export function usePaletteView(session: PaletteSession) {
  const { deps, state, steps, on, rows } = session
  const { getCommands, byId } = rows

  const draws = getPaletteView({
    words: deps.words,
    get commands() {
      return getCommands()
    },
    get byId() {
      return byId.value
    },
    runs: deps.runs,
    lists: deps.lists,
    found: state.results.found,
    known: state.results.known,
    showing: state.results.showing,
    isWorking: state.results.isWorking,
    failureMessage: state.results.failureMessage,
    getStepTitle: steps.naming.getStepTitle,
    getStepLabel: steps.naming.getStepLabel,
  })

  const groups = computed(() => draws.createGroups(steps.here.value, on.value, state.typed.value))

  /** The icon a command declares, and nothing for one that declares none. */
  const findCommandIcon = (id: string): Component | null => byId.value.get(id)?.icon ?? null

  /** Whether the vault is shown aside, and cannot be chosen. */
  const isVaultAside = (vault: Vault): boolean => Boolean(draws.getVaultAside(vault))

  return { groups, findTabKind: draws.findTabKind, findCommandIcon, isVaultAside }
}
