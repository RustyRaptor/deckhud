# Flatpak

Local build and install (needs `org.flatpak.Builder` from Flathub):

```sh
flatpak run org.flatpak.Builder --user --force-clean --install-deps-from=flathub --install \
  --state-dir=.flatpak-builder build-flatpak packaging/flatpak/io.github.RustyRaptor.deckhud.yml
flatpak run io.github.RustyRaptor.deckhud          # asks once to start with Plasma
flatpak run io.github.RustyRaptor.deckhud peek     # toggle the cheat sheet
flatpak run io.github.RustyRaptor.deckhud autostart off
```

## Flathub

The Flathub manifest is this file with the `type: dir` source replaced by the release tag:

```yaml
      - type: git
        url: https://github.com/RustyRaptor/deckhud.git
        tag: vX.Y.Z
        commit: <full sha of the tag>
```

When Go dependencies change, regenerate the `proxy.golang.org` archive sources and the inline
`vendor/modules.txt` (`go mod vendor` prints it) in the manifest.

### Permission rationale (for reviewers)

| Permission | Why |
|---|---|
| `--socket=wayland` only | The overlay is a `wlr-layer-shell` surface, which exists only on Wayland. No X11 fallback is possible |
| `--device=all` | Reads the Steam Deck controller's raw HID reports from `/dev/hidraw*` (read-only, alongside Steam). `--device=input` covers only `/dev/input/event*` |
| `~/.local/share/Steam:ro`, Flatpak Steam's equivalent `:ro` | Reads Steam Input layouts, `logs/controller.txt` (which layout is active) and binding icons. Never writes |
| (portals) | Settings portal for the accent colour, Background portal for autostart |
