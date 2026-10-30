# Spike: Flatpak packaging (2026-10-30)

**Result: viable.** Built with `org.flatpak.Builder` against `org.kde.Platform//6.11`.

| Question | Finding |
|---|---|
| Does KWin let a sandboxed client use `wlr-layer-shell`? | **Yes.** Overlay layer, transparent, above windows, same as on the host |
| LayerShellQt in the runtime? | **No.** Built `invent.kde.org/plasma/layer-shell-qt` tag `v6.7.3` as a module, unmodified (cmake-ninja, `-DBUILD_TESTING=OFF`) |
| `qml` runner in the runtime? | **No** (SDK only). A ~10-line C++ launcher (`QGuiApplication` + `QQmlApplicationEngine`) replaces it |
| Kirigami / QtWebSockets / QtQuick.Shapes in runtime? | Yes |
| Controller access | `--device=all` → `/dev/hidraw2` readable inside the sandbox (report `0x09` streams) |
| Steam files | `--filesystem=~/.local/share/Steam:ro` exposes `logs/controller.txt`, configs, binding icons |
| Loopback WebSocket | Works inside one sandbox instance without `--share=network`. **Separate `flatpak run` instances do not share loopback**, so a CLI → daemon control channel must use a Unix socket under `$XDG_RUNTIME_DIR/app/<id>/` |
| `--filesystem=<abs path>` on `flatpak run` | Did not expose a hidden dir under `~/Projects`; `home:ro` did. `~/.cache` is never exposed |

## Accent colour: Kirigami is not reliable, the portal is

Plasma here derives the accent from a wallpaper **slideshow that changes every ~10 s**, which made single
readings misleading. Sampling host `qml`, sandboxed launcher and `kdeglobals` side by side for 60 s:

- **Neither** process's `Kirigami.Theme` colour followed live changes (both stayed on their start value).
- The sandbox's start value equalled `[Colors:Selection] BackgroundNormal`; the host `qml` runner's did not
  match the file at all.

`org.freedesktop.portal.Settings` works on the host session bus and from sandboxes by default:

```sh
busctl --user call org.freedesktop.portal.Desktop /org/freedesktop/portal/desktop \
  org.freedesktop.portal.Settings ReadOne ss org.freedesktop.appearance accent-color
# v (ddd) 0.941176 0.611765 0.792157
```

and emits `SettingChanged(org.freedesktop.appearance, accent-color, (ddd))` on every slideshow change.
**Decision:** `deckhudd` reads the accent from the portal (fallback: `kdeglobals [General] AccentColor`)
and pushes it to the UI as an `accent` message. Kirigami is no longer used for colour.
