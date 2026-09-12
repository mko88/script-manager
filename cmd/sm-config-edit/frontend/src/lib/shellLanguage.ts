// Which syntax an inline command is written in, from the shell the config
// runs actions with. Mirrors action.Language's shell half in Go, which the
// backend uses for script files; the command editor needs it for a draft
// that hasn't been saved, so it can't ask the backend.
export function shellLanguage(shellBin: string | undefined): string {
  const base = (shellBin ?? '')
    .split(/[\/]/)
    .pop()!
    .toLowerCase()
    .replace(/\.exe$/, '')
  if (base === 'pwsh' || base === 'powershell') return 'powershell'
  if (base === 'cmd') return 'plain'
  return base ? 'shell' : 'plain'
}
