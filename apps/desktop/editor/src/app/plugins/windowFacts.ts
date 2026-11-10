/**
 * What the plugin host reads of the window before the window can say it: the commands of the
 * application and the variables of when that say what is in front. The host is made first with this
 * state; the commands part of the window provides the source once it exists.
 */
import { shallowRef } from 'vue'
import type { WhenValues } from '@numen/plugin-loader'
import type { Command } from '@/entities/command'

/** Where the facts come from once the window has made its commands. */
export interface WindowFactsSource {
  readonly getAppCommands: () => readonly Command[]
  readonly getValues: () => WhenValues
}

export function createWindowFacts() {
  const source = shallowRef<WindowFactsSource>({ getAppCommands: () => [], getValues: () => ({}) })
  return {
    getAppCommands: (): readonly Command[] => source.value.getAppCommands(),
    getValues: (): WhenValues => source.value.getValues(),
    provide: (next: WindowFactsSource): void => {
      source.value = next
    },
  }
}

export type WindowFacts = ReturnType<typeof createWindowFacts>
