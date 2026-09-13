import { askConfirm } from '@shared/confirm'
import { t } from '../messages'

// Replaces window.confirm: same question, asked in the app's own dialog.
// Awaiting it is the difference — callers have to be async.
export function confirmAsk(message: string): Promise<boolean> {
  return askConfirm({
    title: t('confirm.title'),
    message,
    confirmLabel: t('confirm.okButton'),
    cancelLabel: t('confirm.cancelButton'),
  })
}
