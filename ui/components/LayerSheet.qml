pragma ComponentBehavior: Bound
import QtQuick

// Cheat sheet: every bound control in the current action set and layers,
// left-hand controls on the left, right-hand on the right.
Item {
    id: sheetView

    property color accent: "#9a93e3"
    property var sheet: ({ set: "", controls: [] })
    property bool active: false

    readonly property string title: [sheet.set].concat(sheet.layers ?? []).join(" › ")
    readonly property var leftGlyphs: ["L2", "L1", "L4", "L5", "View", "LS", "D-pad", "LPad"]
    readonly property var leftControls: (sheet.controls ?? []).filter(c => leftGlyphs.includes(c.glyph))
    readonly property var rightControls: (sheet.controls ?? []).filter(c => !leftGlyphs.includes(c.glyph))
    readonly property real fontSize: Math.max(11, height * 0.017)

    function show(newSheet, duration) {
        sheet = newSheet
        active = true
        if (duration > 0) {
            hideTimer.interval = duration
            hideTimer.restart()
        } else {
            hideTimer.stop()
        }
    }
    function hide() {
        hideTimer.stop()
        active = false
    }
    function toggle(newSheet) {
        if (active)
            hide()
        else
            show(newSheet, 0)
    }

    function entryText(entry) {
        const prefixes = { Long_Press: "hold", Double_Press: "2×", Start_Press: "down", Release_Press: "up",
                           Soft_Press: "soft", chord: "chord" }
        const directions = { dpad_north: "▲", dpad_east: "▶", dpad_south: "▼", dpad_west: "◀",
                             click: "click", edge: "edge", touch: "touch" }
        const keys = (entry.keys ?? []).join("+")
        const parts = []
        const input = entry.input ?? ""
        if (input.startsWith("touch_menu_button_"))
            parts.push(input.slice("touch_menu_button_".length))
        else if (input !== "")
            parts.push(directions[input] ?? input.replace(/_/g, " "))
        if (prefixes[entry.activator])
            parts.push(prefixes[entry.activator])
        parts.push(entry.label ? entry.label + "  (" + keys + ")" : keys)
        return parts.join("  ")
    }
    function modeText(mode) {
        const names = { absolute_mouse: "Mouse", joystick_mouse: "Mouse", mouse_joystick: "Mouse",
                        radial_menu: "Radial menu", touch_menu: "Touch menu", scrollwheel: "Scroll wheel",
                        joystick_move: "Joystick", "2dscroll": "2D scroll", flickstick: "Flick stick",
                        four_buttons: "Buttons", switches: "Buttons", dpad: "D-pad", trigger: "Trigger",
                        single_button: "Button" }
        if (names[mode]) return names[mode]
        const words = mode.replace(/_/g, " ")
        return words.charAt(0).toUpperCase() + words.slice(1)
    }

    opacity: active ? 1 : 0
    visible: opacity > 0
    Behavior on opacity { OpacityAnimator { duration: 150 } }

    Timer {
        id: hideTimer
        onTriggered: sheetView.active = false
    }

    Rectangle {
        anchors.fill: parent
        color: Qt.rgba(0.02, 0.02, 0.04, 0.55)
    }

    Text {
        id: heading
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.top: parent.top
        anchors.topMargin: sheetView.height * 0.03
        text: sheetView.title
        color: "white"
        font.pixelSize: sheetView.fontSize * 1.6
        font.weight: Font.Bold
        style: Text.Outline
        styleColor: Qt.alpha(sheetView.accent, 0.6)
    }

    component ControlColumn: Column {
        id: column
        property var controls: []
        spacing: sheetView.fontSize * 0.5

        Repeater {
            model: column.controls
            delegate: Rectangle {
                id: card
                required property var modelData
                width: Math.min(sheetView.width * 0.4, cardRow.implicitWidth + 20)
                height: cardRow.implicitHeight + 12
                radius: 8
                color: card.modelData.changed ? Qt.alpha(sheetView.accent, 0.35) : Qt.rgba(0.05, 0.05, 0.08, 0.7)
                border.color: Qt.alpha(sheetView.accent, card.modelData.changed ? 1 : 0.5)
                border.width: card.modelData.changed ? 2 : 1

                Row {
                    id: cardRow
                    x: 10
                    y: 6
                    spacing: 10

                    ButtonGlyph {
                        button: card.modelData.glyph
                        accent: sheetView.accent
                        implicitHeight: sheetView.fontSize * 2
                    }
                    Column {
                        Text {
                            visible: card.modelData.entries.length === 0 || card.modelData.entries[0].input !== undefined
                            text: sheetView.modeText(card.modelData.mode)
                            color: sheetView.accent
                            font.pixelSize: sheetView.fontSize * 0.85
                            font.weight: Font.DemiBold
                        }
                        Repeater {
                            model: card.modelData.entries
                            delegate: Text {
                                required property var modelData
                                width: Math.min(implicitWidth, sheetView.width * 0.4 - 80)
                                elide: Text.ElideRight
                                text: sheetView.entryText(modelData)
                                color: "white"
                                font.pixelSize: sheetView.fontSize
                            }
                        }
                    }
                }
            }
        }
    }

    ControlColumn {
        anchors.left: parent.left
        anchors.leftMargin: sheetView.width * 0.03
        anchors.top: heading.bottom
        anchors.topMargin: sheetView.fontSize
        controls: sheetView.leftControls
    }
    ControlColumn {
        anchors.right: parent.right
        anchors.rightMargin: sheetView.width * 0.03
        anchors.top: heading.bottom
        anchors.topMargin: sheetView.fontSize
        controls: sheetView.rightControls
    }
}
