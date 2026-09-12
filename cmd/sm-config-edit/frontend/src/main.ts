import './style.css'
import App from './App.svelte'
import { setMessageOverride } from './messages'
import { GetMessages, GetTheme } from '../wailsjs/go/configedit/App.js'
import { initTheme, syncTheme } from '@shared/theme'

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
