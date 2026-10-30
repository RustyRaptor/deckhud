import QtQuick

// Everything deckhud draws, driven by daemon messages via handle().
Item {
    id: hud

    property color accent: "#9a93e3"
    Behavior on accent { ColorAnimation { duration: 400 } }
    property string iconDir: ""
    property bool quietNextState: false
    property var lastSheet: null
    property int sheetTimeout: 2500
    readonly property alias radial: radialHud
    readonly property alias toasts: chordToasts
    readonly property alias banner: stateBanner
    readonly property alias sheet: layerSheet

    function handleText(text) {
        let msg
        try {
            msg = JSON.parse(text)
        } catch (e) {
            console.warn("deckhud: bad message", text)
            return
        }
        handle(msg)
    }

    function handle(msg) {
        switch (msg.type) {
        case "accent":
            if (/^#[0-9a-fA-F]{6}$/.test(msg.color ?? ""))
                accent = msg.color
            break
        case "config":
            if (msg.iconDir)
                iconDir = msg.iconDir
            break
        case "hello":
            quietNextState = true
            break
        case "radial":
            handleRadial(msg.radial)
            break
        case "chord":
            if (!radialHud.active)
                chordToasts.push(msg.chord)
            break
        case "state": {
            const text = [msg.state.set].concat(msg.state.layers ?? []).join(" › ")
            if (msg.sheet)
                lastSheet = msg.sheet
            if (quietNextState) {
                stateBanner.text = text
                quietNextState = false
            } else if (msg.sheet && !radialHud.active) {
                layerSheet.show(msg.sheet, sheetTimeout)
            } else {
                stateBanner.show(text)
            }
            break
        }
        case "peek":
            if (layerSheet.active)
                layerSheet.hide()
            else if (lastSheet)
                layerSheet.show(lastSheet, 0)
            break
        }
    }

    function handleRadial(r) {
        switch (r.phase) {
        case "open":
            chordToasts.clear()
            layerSheet.hide()
            radialHud.open(r.items ?? [], r.title ?? "", r.index)
            break
        case "move":
            radialHud.select(r.index)
            break
        case "fire":
            radialHud.fire(r.index)
            break
        case "cancel":
            radialHud.cancel()
            break
        }
    }

    LayerSheet {
        id: layerSheet
        anchors.fill: parent
        accent: hud.accent
    }
    RadialHud {
        id: radialHud
        anchors.fill: parent
        accent: hud.accent
        iconDir: hud.iconDir
    }
    ChordToasts {
        id: chordToasts
        anchors.fill: parent
        accent: hud.accent
    }
    StateBanner {
        id: stateBanner
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.top: parent.top
        anchors.topMargin: parent.height * 0.05
        accent: hud.accent
    }
}
