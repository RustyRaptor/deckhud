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

Not submitted. Flathub's [Generative AI policy](https://docs.flathub.org/docs/for-app-authors/requirements#generative-ai-policy)
does not allow AI-generated or AI-assisted content in Flathub manifests, or AI-prepared submission
pull requests, and requires disclosing AI-generated application code (much of deckhud was written
with Claude). It also expects [meaningful development history](https://docs.flathub.org/docs/for-app-authors/requirements#insufficient-development-history).
A Flathub submission, if any, has to be authored by the maintainer under those rules.

The manifest here is for local builds and self-hosted distribution.

## Permission rationale

| Permission | Why |
|---|---|
| `--socket=wayland` only | The overlay is a `wlr-layer-shell` surface, which exists only on Wayland. No X11 fallback is possible |
| `--device=all` | Reads the Steam Deck controller's raw HID reports from `/dev/hidraw*` (read-only, alongside Steam). `--device=input` covers only `/dev/input/event*` |
| `~/.local/share/Steam:ro`, Flatpak Steam's equivalent `:ro` | Reads Steam Input layouts, `logs/controller.txt` (which layout is active) and binding icons. Never writes |
| (portals) | Settings portal for the accent colour, Background portal for autostart |
