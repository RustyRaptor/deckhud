# Spike: layer-shell overlay with stock SteamOS QML (2026-10-30)

**Result: works.** Plasma 6.7.3 / Qt 6.11.1, run with `/usr/lib/qt6/bin/qml spike.qml`, no compiled code.

```qml
import QtQuick
import QtQuick.Window
import org.kde.kirigami as Kirigami
import org.kde.layershell as LayerShell

Window {
    visible: true
    color: "transparent"
    flags: Qt.FramelessWindowHint | Qt.WindowTransparentForInput | Qt.WindowDoesNotAcceptFocus
    LayerShell.Window.scope: "deckhud"
    LayerShell.Window.layer: LayerShell.Window.LayerOverlay
    LayerShell.Window.anchors: LayerShell.Window.AnchorTop | LayerShell.Window.AnchorBottom
                             | LayerShell.Window.AnchorLeft | LayerShell.Window.AnchorRight
    LayerShell.Window.keyboardInteractivity: LayerShell.Window.KeyboardInteractivityNone
    LayerShell.Window.exclusionZone: -1
    LayerShell.Window.activateOnShow: false
    // ...
}
```

## Findings

| Topic | Finding |
|---|---|
| Transparency | Per-pixel alpha composites correctly over windows and wallpaper |
| Placement | With no `screen` set, the surface went to the Deck's built-in panel (not the external monitor) |
| Accent colour | `Kirigami.Theme.highlightColor` = kdeglobals `[Colors:Selection] BackgroundNormal`, which Plasma derives from `[General] AccentColor`. With `accentColorFromWallpaper=true` and a slideshow, the accent changes with every wallpaper; the HUD should animate colour changes |
| Platform theme | Same colours with/without `QT_QPA_PLATFORMTHEME=kde` / `QT_QUICK_CONTROLS_STYLE=org.kde.desktop`; Kirigami's platform plugin finds KDE's colours by itself |
| Theme timing | Colours read as `#000000` in `Component.onCompleted`; they are valid after the first frame. Use bindings, never one-shot reads |
| Logging | Qt logs to the **journal** when stderr isn't a TTY. Use `QT_FORCE_STDERR_LOGGING=1` in tests and units |
| Legibility | Busy wallpapers drown a pure accent tint: needs a dark translucent backdrop + accent rim/highlight |

## Still to confirm by hand
- Click-through and touch pass-through (`Qt.WindowTransparentForInput` → empty input region)
- Stays above fullscreen windows (games, video)
- Which output to use with an external display attached (follow pointer? always internal?)
