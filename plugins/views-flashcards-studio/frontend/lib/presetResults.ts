import type { Result } from '@numen/plugin-sdk'
import type { CardsFailure, ErrorCode } from '../types'
import type { Preset, SettingsBounds } from './presets'

/** A preset as a read found it, and how far each of its settings goes. */
interface ReadPreset {
  readonly preset: Preset | null
  readonly at: string
  readonly bounds: SettingsBounds
}

/** What reading a preset returned. */
export type PresetReadResult = Result<ReadPreset, ErrorCode>

/** What writing a preset returned. */
export type PresetWriteResult = Result<{ readonly at: string }, CardsFailure>
