import './style.css'
import App from './App.svelte'
import { setMessageOverride } from './messages'
import { GetMessages } from '../wailsjs/go/configedit/App.js'
import { initTheme } from '@shared/theme'

initTheme()

async function bootstrap() {
  try {
    setMessageOverride(await GetMessages())
  } catch {
    // Missing/invalid override file — t() falls back to compiled defaults.
  }

  return new App({
    target: document.getElementById('app')!,
  })
}

export default bootstrap()
