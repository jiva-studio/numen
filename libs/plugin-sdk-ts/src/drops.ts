import type { Disposable } from './types'

// DroppedData is what a drag from outside the window carries: the data under each type the drag
// names, such as `text/uri-list` for addresses dragged out of a browser.
export type DroppedData = Readonly<Record<string, string>>

// DropRegistration declares the dropped data a plugin takes, by the types a drag names it under,
// wherever it is dropped: a view takes a drop without knowing which plugin takes it.
export interface DropRegistration {
  // The types of dropped data the plugin takes, such as `text/uri-list`.
  readonly types: readonly string[]
  // Takes what was dropped into a folder of the vault, the root being empty. Answers the paths of
  // what it made.
  drop(folder: string, data: DroppedData): Promise<readonly string[]>
}

// DropHost is how a view hands a drop to the plugin that takes it.
export interface DropHost {
  // Whether a plugin takes data of one of the types.
  canDrop(types: readonly string[]): boolean
  // Hands what was dropped into the folder to the first plugin that takes one of its types, and
  // answers the paths of what it made; none where no plugin takes it.
  drop(folder: string, data: DroppedData): Promise<readonly string[]>
}

// DropRegistry keeps the drops plugins register and hands each drop to the one that takes it.
export interface DropRegistry extends DropHost {
  registerDropHandler(pluginId: string, registration: DropRegistration): Disposable
}
