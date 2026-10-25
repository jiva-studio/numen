/**
 * The settings file read and written by key path: a value as the type a field
 * needs, and a value or a model written back.
 */
import type { Model, SettingEdit } from './configuration'
import { write } from './write'

/** Where the settings stand, and where a change to them goes. */
export interface SettingSource {
  getSetting(path: readonly string[]): unknown
  getModels(path: readonly string[]): readonly Model[]
  write(content: readonly SettingEdit[]): void
}

export interface SettingFields {
  /** The text at a setting, and nothing for any other value. */
  getString(path: readonly string[]): string
  /** Whether a setting is on. */
  isEnabled(path: readonly string[]): boolean
  /** The number at a setting, and the fallback for any other value. */
  getNumber(path: readonly string[]): number | null
  getNumber(path: readonly string[], fallback: number): number
  /** The models a setting that names one can be set to. */
  getModels(path: readonly string[]): readonly Model[]
  /** A value written at a setting. */
  setValue(path: readonly string[], value: unknown): void
  /** A model chosen by name: its own writes where it is a preset, else the name itself. */
  chooseModel(path: readonly string[], name: string): void
}

/** The source is asked for on every call, so a source that changes is followed. */
export function createSettingFields(getSource: () => SettingSource): SettingFields {
  const getString = (path: readonly string[]): string => {
    const value = getSource().getSetting(path)
    return typeof value === 'string' ? value : ''
  }

  const isEnabled = (path: readonly string[]): boolean => getSource().getSetting(path) === true

  function getNumber(path: readonly string[]): number | null
  function getNumber(path: readonly string[], fallback: number): number
  function getNumber(path: readonly string[], fallback: number | null = null): number | null {
    const value = getSource().getSetting(path)
    return typeof value === 'number' ? value : fallback
  }

  const getModels = (path: readonly string[]): readonly Model[] => getSource().getModels(path)

  const setValue = (path: readonly string[], value: unknown): void => {
    getSource().write([{ at: path, value: write(value) }])
  }

  const chooseModel = (path: readonly string[], name: string): void => {
    const model = getModels(path).find((one) => one.name === name)
    if (model) getSource().write(model.writes)
    else setValue(path, name)
  }

  return { getString, isEnabled, getNumber, getModels, setValue, chooseModel }
}
