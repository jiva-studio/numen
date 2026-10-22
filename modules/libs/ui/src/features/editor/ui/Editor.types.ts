import type { Extension } from '@codemirror/state'
import type { EditorChange } from '../lib/change'

export interface EditorProps {
  /** Marks are drawn as what they mean. Off, the text is shown as written. */
  isLive?: boolean | undefined
  /**
   * What the whole document is written in, by the name a fence would use. A
   * document naming none is markdown, and one naming a language is set in the
   * face code is set in.
   */
  language?: string | undefined
  isReadonly?: boolean | undefined
  placeholder?: string | undefined
  /** What it is announced as. What has been typed here is no name for it. */
  name?: string | undefined
  /**
   * How to leave, read out on arrival. Tab is the editor's while a person is
   * writing, so the way back out has to be said or nobody finds it.
   */
  keys?: string | undefined
  /** A change being made to this text by something other than the reader. */
  change?: EditorChange | null | undefined
  /** What an address in the text becomes before the window loads it. */
  resolve?: ((address: string) => string) | undefined
  /** More the caller draws into this editor. */
  extensions?: Extension | undefined
}
