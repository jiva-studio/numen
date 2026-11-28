export interface DialogProps {
  label: string
  hint?: string
  role?: 'dialog' | 'alertdialog'
  tone?: 'danger' | 'default'
  describedBy?: string
  step?: string
  size?: 'sm' | 'md' | 'lg'
  isForm?: boolean
}

export interface DialogPanelProps extends DialogProps {
  scope: 'window' | 'host'
}

export interface DialogEvents {
  (e: 'submit'): void
  (e: 'cancel'): void
}
