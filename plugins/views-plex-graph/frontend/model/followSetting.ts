/** One setting of the plugin, parsed and followed while the plugin is active. */
import { shallowRef, type Ref } from 'vue'
import type { JsonValue, PluginContext } from '@numen/plugin-sdk'

export function followSetting<T>(
  context: Pick<PluginContext, 'settings'>,
  key: string,
  parse: (value: JsonValue | undefined) => T,
): Ref<T> {
  const value = shallowRef(parse(context.settings.get(key)))
  context.settings.onChange(key, (next) => {
    value.value = parse(next)
  })
  return value
}

/** The settings of a context that holds none, so every one is at its default. */
export const NO_SETTINGS: Pick<PluginContext, 'settings'> = {
  settings: { get: () => undefined, onChange: () => ({ dispose: () => undefined }) },
}
