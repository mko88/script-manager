import './style.css'
import App from './App.svelte'
import { setMessageOverride } from './messages'
import { GetMessages, GetTheme, Log } from '../wailsjs/go/gui/App.js'
import { initTheme, syncTheme } from '@shared/theme'

// Anything the interface throws goes to the same log file the Go side
// writes, where an exception and the call that preceded it sit together.
// A frozen window can't show a console, so this is the only account of
// what happened.
function reportToLog(what: string, detail: unknown) {
  const err = detail as { message?: string; stack?: string } | undefined
  const text = err?.stack || err?.message || String(detail)
  try {
    Log(`${what}: ${text}`)
  } catch {
  }
}

window.addEventListener('error', (e) => reportToLog('window.error', e.error ?? e.message))
window.addEventListener('unhandledrejection', (e) => reportToLog('unhandledrejection', e.reason))

const originalConsoleError = console.error.bind(console)
console.error = (...args: unknown[]) => {
  reportToLog('console.error', args.map((a) => (a instanceof Error ? a.stack || a.message : String(a))).join(' '))
  originalConsoleError(...args)
}

initTheme()

async function bootstrap() {
  try {
    setMessageOverride(await GetMessages())
  } catch {
  }
  await syncTheme(GetTheme)

  return new App({
    target: document.getElementById('app')!,
  })
}

export default bootstrap()
