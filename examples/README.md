# Example configs

## PIN-locked secrets

`pin-secrets.yaml` (bash) and `pin-secrets-win.yaml` (PowerShell) are the same
demo config for the [PIN-locked values](../README.md#pin-locked-values)
feature, differing only in `shell:` and the action commands.

**The PIN is `demo1234`.**

Load one with any of the three apps:

```bash
./bin/script-manager      -config examples/pin-secrets.yaml
./bin/script-manager-gui  -config examples/pin-secrets.yaml
./bin/sm-config-edit      -config examples/pin-secrets.yaml
```

```powershell
.\bin\script-manager-gui.exe -config examples\pin-secrets-win.yaml
```

What it contains:

- A global `vault_token` in `env:`, locked — so every item inherits a locked
  value.
- `demo-database` and `demo-api`, each with its own locked `db_password`.
- `demo-unlocked`, whose password is plain text, for comparison.
- Three actions: **Show the decrypted values (needs PIN)** and **Pretend to
  connect (needs PIN)** both have `requiresPin: true`, while **Show the same
  values (no PIN)** runs the same command without it.

What to try:

1. Open an item before entering the PIN — the Details pane shows `(locked)`
   where a locked value would be.
2. Run **Show the same values (no PIN)** first. It runs with no prompt, and
   prints `db_password=[]` and `vault_token=[]` — the locked variables are
   never set for an action that doesn't ask for them.
3. Run **Show the decrypted values (needs PIN)**. This one prompts; enter
   `demo1234` and the script prints the real passwords.
4. Run **Pretend to connect** — no prompt this time; the unlock lasts until the
   app closes.
5. Open the file in a text editor. Only `sm-enc:v1:…` ciphertext is there.
6. In `sm-config-edit`, click the padlock beside `db_password` to reveal or
   re-lock it (a locked field is read-only until you do), and find the
   **Requires PIN** checkbox on each action.

The locked values here are `correct-horse-battery-staple`, its API equivalent,
and a fake Vault token — all throwaway strings committed on purpose so the demo
works out of the box. A real config would not be committed with its ciphertext
in a public repository, and `demo1234` is exactly the kind of PIN the main
README tells you not to use.
