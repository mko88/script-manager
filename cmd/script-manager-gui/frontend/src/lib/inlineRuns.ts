// Tracks every "Run here" (inline) execution for the session: one entry per
// item/action pair that's ever been run, keyed by inlineKey — not just the
// single currently-viewed one, so a run started on one action keeps going
// (and stays pollable) after switching to a different action, and switching
// back shows however far it's gotten (or its finished result) instead of
// losing track of it.
import { writable, get } from 'svelte/store'
import { flash } from '@shared/toast'
import { t } from '../messages'
import { GetInlineStatus, RunActionInline, CancelInlineAction } from '../../wailsjs/go/gui/App.js'

export type InlineState = {
  itemIndex: number
  actionIndex: number
  output: string
  running: boolean
}

export const inlineStates = writable<Record<string, InlineState>>({})

export function inlineKey(itemIndex: number, actionIndex: number): string {
  return `${itemIndex}:${actionIndex}`
}

function setInlineState(itemIndex: number, actionIndex: number, state: Omit<InlineState, 'itemIndex' | 'actionIndex'>) {
  inlineStates.update((states) => ({
    ...states,
    [inlineKey(itemIndex, actionIndex)]: { itemIndex, actionIndex, ...state },
  }))
}

// How the frontend gets a live-updating view of an inline run: polling
// GetInlineStatus on a short timer, not a pushed event — this app's other
// bound methods are all plain request/response calls, and that's the shape
// that's held up reliably here (see App.svelte's scrollInlineOutputToEnd doc
// comment for the actual bug that made earlier streaming attempts look
// unreliable — it wasn't Wails or this call shape at all).
//
// A poll loop keeps going until its own run finishes, regardless of whether
// the user is still looking at that action — that's what makes "switch away,
// switch back" work: inlineStates already has whatever this loop has
// captured by the time the user returns, instead of the loop having given up
// and stopped tracking it. onUpdate fires after every poll tick so the
// caller can run its scroll-into-view side effect for the on-screen action.
const INLINE_POLL_INTERVAL_MS = 300

async function pollInlineStatus(itemIndex: number, actionIndex: number, onUpdate: (itemIndex: number, actionIndex: number) => void) {
  for (;;) {
    const status = await GetInlineStatus(itemIndex, actionIndex)
    setInlineState(itemIndex, actionIndex, { output: status.output, running: status.running })
    onUpdate(itemIndex, actionIndex)
    if (status.running) {
      await new Promise((resolve) => setTimeout(resolve, INLINE_POLL_INTERVAL_MS))
      continue
    }
    if (status.errMsg) flash(t('toast.runFailed', { error: status.errMsg }))
    return
  }
}

// Starts an inline run and its poll loop. Starting the same pair again while
// it's still running is rejected (a no-op); different pairs may overlap.
export async function startInlineRun(itemIndex: number, actionIndex: number, onUpdate: (itemIndex: number, actionIndex: number) => void) {
  if (get(inlineStates)[inlineKey(itemIndex, actionIndex)]?.running) return
  setInlineState(itemIndex, actionIndex, { output: '', running: true })
  try {
    await RunActionInline(itemIndex, actionIndex)
    pollInlineStatus(itemIndex, actionIndex, onUpdate)
  } catch (err) {
    setInlineState(itemIndex, actionIndex, { output: '', running: false })
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
