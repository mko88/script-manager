import { writable, get } from 'svelte/store'
import { flash } from '@shared/toast'
import { t } from '../messages'
import { GetInlineStatus, RunActionInline, CancelInlineAction } from '../../wailsjs/go/gui/App.js'

export type InlineState = {
  itemIndex: number
  actionIndex: number
  output: string
  running: boolean
  exitCode: number | null
}

export const inlineStates = writable<Record<string, InlineState>>({})

// Bumped when the config changes. Polls started against the old config must
// not write their last status back into the cleared store.
let generation = 0

export function resetInlineRuns() {
  generation += 1
  inlineStates.set({})
}

export function inlineKey(itemIndex: number, actionIndex: number): string {
  return `${itemIndex}:${actionIndex}`
}

function setInlineState(itemIndex: number, actionIndex: number, state: Omit<InlineState, 'itemIndex' | 'actionIndex'>) {
  inlineStates.update((states) => ({
    ...states,
    [inlineKey(itemIndex, actionIndex)]: { itemIndex, actionIndex, ...state },
  }))
}

const INLINE_POLL_INTERVAL_MS = 300

async function pollInlineStatus(itemIndex: number, actionIndex: number) {
  const gen = generation
  for (;;) {
    const status = await GetInlineStatus(itemIndex, actionIndex)
    if (gen !== generation) return
    setInlineState(itemIndex, actionIndex, {
      output: status.output,
      running: status.running,
      exitCode: status.running ? null : status.exitCode,
    })
    if (status.running) {
      await new Promise((resolve) => setTimeout(resolve, INLINE_POLL_INTERVAL_MS))
      continue
    }
    if (status.errMsg) flash(t('toast.runFailed', { error: status.errMsg }))
    return
  }
}

export async function startInlineRun(itemIndex: number, actionIndex: number) {
  if (get(inlineStates)[inlineKey(itemIndex, actionIndex)]?.running) return
  const gen = generation
  setInlineState(itemIndex, actionIndex, { output: '', running: true, exitCode: null })
  try {
    await RunActionInline(itemIndex, actionIndex)
    if (gen !== generation) return
    pollInlineStatus(itemIndex, actionIndex)
  } catch (err) {
    if (gen !== generation) return
    setInlineState(itemIndex, actionIndex, { output: '', running: false, exitCode: null })
    flash(t('toast.runFailed', { error: String(err) }))
  }
}

export async function cancelInlineRun(itemIndex: number, actionIndex: number) {
  try {
    await CancelInlineAction(itemIndex, actionIndex)
  } catch (err) {
    flash(t('toast.cancelFailed', { error: String(err) }))
  }
}
