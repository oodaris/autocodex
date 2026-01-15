# Plugins

autocodex plugins are external processes described by a `plugin.yaml` or `plugin.json` manifest. The host launches the plugin, performs a JSON handshake, and then issues capability calls.

## Sample plugin
A sample plugin lives in `plugins/sample-summarizer/`.

Build it:
```bash
go build -o plugins/sample-summarizer/sample-summarizer ./plugins/sample-summarizer
```

Run it via CLI once the plugin host command is wired:
```bash
autocodex plugins run --name sample-summarizer --capability summarize --input '{"text":"hello world"}'
```

## Manifest
See `docs/contracts/plugin-protocol.md` for the protocol spec.
