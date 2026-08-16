# Desktop development and release

The desktop application uses Wails v2 and the shared React application in
`../../web`. API requests are handled in-process through the Wails asset server;
the desktop application does not open a loopback HTTP port.

The first screen is the visual configuration workspace. Switch to Chat from the
left navigation. Chat conversations are stored only in the local WebView
storage (up to 50 conversations / approximately 4 MiB) and are not uploaded to
the server. The connection Access Key remains session-only.

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0
cd cmd/desktop
wails dev
wails build -clean
```

On first launch the app creates a private configuration file under the current
user's configuration directory. Pass an explicit config file as the first
argument when developing against an existing setup.

Production artifacts are written to `build/bin/`. Configuration fields and
SQLite behavior are documented in [`../../docs/configuration-reference.md`](../../docs/configuration-reference.md).
