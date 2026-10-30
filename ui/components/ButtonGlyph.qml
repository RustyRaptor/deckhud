import QtQuick

// An original, vector glyph for one Steam Deck control.
Item {
    id: glyph

    property string button
    property color accent: "#9a93e3"
    property color foreground: "white"

    readonly property string shape: {
        if (["A", "B", "X", "Y"].includes(button)) return "face"
        if (["L1", "R1"].includes(button)) return "bumper"
        if (["L2", "R2"].includes(button)) return "trigger"
        if (["L4", "L5", "R4", "R5"].includes(button)) return "back"
        if (["Up", "Down", "Left", "Right", "D-pad"].includes(button)) return "dpad"
        if (["L3", "R3", "LS", "RS"].includes(button)) return "stick"
        if (["LPadClick", "RPadClick", "LPad", "RPad"].includes(button)) return "pad"
        return "system"
    }
    readonly property string glyphText: {
        const symbols = { Up: "▲", Down: "▼", Left: "◀", Right: "▶", View: "⧉", Menu: "☰",
                          Steam: "STEAM", QAM: "•••", "D-pad": "✛",
                          LPadClick: "LP", RPadClick: "RP", LPad: "LP", RPad: "RP" }
        return symbols[button] ?? button
    }

    implicitHeight: 36
    implicitWidth: ["face", "dpad", "stick", "pad"].includes(shape) ? implicitHeight
                 : Math.max(implicitHeight * 1.3, label.implicitWidth + 16)

    Accessible.role: Accessible.StaticText
    Accessible.name: button

    Rectangle {
        id: body
        anchors.fill: parent
        radius: glyph.shape === "face" || glyph.shape === "stick" ? height / 2
              : glyph.shape === "trigger" ? height * 0.45
              : glyph.shape === "bumper" ? height * 0.3
              : 6
        color: Qt.rgba(1, 1, 1, 0.14)
        border.color: glyph.accent
        border.width: 2
    }
    Rectangle {
        visible: glyph.shape === "stick" || glyph.shape === "pad"
        anchors.centerIn: parent
        width: parent.height * 0.62
        height: width
        radius: glyph.shape === "stick" ? width / 2 : 4
        color: "transparent"
        border.color: Qt.alpha(glyph.accent, 0.6)
        border.width: 1
    }
    Text {
        id: label
        anchors.centerIn: parent
        text: glyph.glyphText
        color: glyph.foreground
        font.bold: true
        font.pixelSize: glyph.height * (glyph.glyphText.length > 2 ? 0.32 : 0.45)
    }
}
