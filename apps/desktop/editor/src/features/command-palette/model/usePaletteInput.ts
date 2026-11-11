/**
 * The single input field of the window: what it draws, the commands while they are open and the
 * search otherwise, and what typing in it, choosing a row, dismissing and going back mean.
 */
import { computed } from 'vue'
import { reportFailure, runInBackground } from '@numen/ui'
import type { CommandDeps } from '@/entities/command'
import { isCommandPrefixTyped } from '../lib/commands/commands'
import { runInvocation } from './handlers/handlers'
import type { Commands } from './palette/palette'
import type { ResultChoice, SearchState } from './search/search'

export interface PaletteInputDeps {
  readonly commands: Commands
  readonly search: SearchState
  readonly doing: CommandDeps
  readonly runChoice: (choice: ResultChoice) => void
  readonly findPlaceholder: () => string
}

export function usePaletteInput(deps: PaletteInputDeps) {
  const { commands, search } = deps

  /** What the palette draws: the commands while they are open, the search under. */
  const field = computed(() =>
    commands.palette.open.value
      ? {
          open: true,
          isBusy: false,
          typed: commands.palette.typed.value,
          groups: commands.view.groups.value,
          crumb: commands.navigation.crumb.value,
          step: commands.navigation.step.value,
          opensOn: commands.navigation.opensOn.value,
          placeholder: commands.navigation.placeholder.value,
        }
      : {
          open: search.open.value,
          isBusy: search.status.isBusy.value,
          typed: search.typed.value,
          groups: search.groups.value,
          crumb: '',
          step: '',
          opensOn: '',
          placeholder: deps.findPlaceholder(),
        },
  )

  function type(text: string): void {
    if (commands.palette.open.value) {
      runInBackground(commands.palette.setTyped(text), reportFailure)
      return
    }
    if (!isCommandPrefixTyped(search.typed.value, text)) {
      runInBackground(search.setTyped(text), reportFailure)
      return
    }
    search.setOpen(false)
    commands.palette.setOpen(true)
  }

  function choose(item: string, action: string): void {
    if (commands.palette.open.value) {
      const invocation = commands.palette.chooseItem(item, action)
      if (!invocation) return
      commands.palette.setOpen(false)
      runInBackground(runInvocation(invocation, deps.doing), reportFailure)
      return
    }
    const choice = search.chooseResult(item, action)
    search.setOpen(false)
    if (choice) deps.runChoice(choice)
  }

  function dismiss(): void {
    if (commands.palette.open.value) commands.navigation.leaveStep()
    else search.setOpen(false)
  }

  function goBack(): void {
    if (commands.palette.open.value && commands.navigation.goBack()) search.setOpen(true)
  }

  return { field, actions: { type, choose, dismiss, goBack } }
}
