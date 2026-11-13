/**
 * What a settings row is drawn from. detail is the one line under the title; details is the full
 * text the info button opens. focus is a request to scroll the row into view, a new object each time.
 */
export interface SettingRowProps {
  at: string
  name: string
  detail: string
  details?: string | undefined
  isMarked?: boolean
  focus?: object | null
  isChanged?: boolean
}
