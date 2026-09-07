# Desktop development and release

The desktop application uses Wails v2 and the shared React application in
`../../web`. UI and chat requests can use the in-process Wails bridge. The app
also starts the same API gateway on `127.0.0.1:<server_port>` (port `9090` by
default), allowing local clients such as Codex and zcode to use
`http://127.0.0.1:9090/v1`. Closing the app shuts down that gateway and releases
the port.

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

If the configured port is already occupied, the desktop UI can still open but
external local clients cannot reach its gateway. Stop the conflicting process
or change `server_port`, then restart the app.
