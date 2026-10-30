import QtQuick
import "../../ui/components"

// Renders store screenshots offscreen:
//   QML_XHR_ALLOW_FILE_READ=1 QT_QPA_PLATFORM=offscreen qml packaging/screenshots/render.qml -- <out-dir>
Window {
    id: win
    width: 1280
    height: 800
    visible: true

    readonly property string outDir: Qt.application.arguments[Qt.application.arguments.length - 1]
    property var steps: []

    function load(name) {
        const xhr = new XMLHttpRequest()
        xhr.open("GET", Qt.resolvedUrl(name), false)
        xhr.send()
        return JSON.parse(xhr.responseText)
    }

    Rectangle {
        anchors.fill: parent
        gradient: Gradient {
            GradientStop { position: 0; color: "#3a4a63" }
            GradientStop { position: 1; color: "#1d2433" }
        }
        Grid {
            anchors.centerIn: parent
            columns: 8
            spacing: 60
            Repeater {
                model: 40
                Rectangle { width: 90; height: 60; radius: 8; color: Qt.rgba(1, 1, 1, 0.05) }
            }
        }
    }

    Hud {
        id: hud
        anchors.fill: parent
        accent: "#9a93e3"
        sheetTimeout: 100000
        toasts.timeout: 100000
    }

    Image {
        id: icon
        visible: false
        source: "../../data/io.github.RustyRaptor.deckhud.svg"
        sourceSize: Qt.size(256, 256)
    }

    function shoot(item, name, next) {
        item.grabToImage(r => { r.saveToFile(outDir + "/" + name); next() })
    }

    Timer {
        interval: 600
        running: true
        onTriggered: {
            const radial = win.load("radial.json")
            radial.radial.index = 2
            hud.handle(radial)
            Qt.callLater(() => wait.start())
        }
    }
    Timer {
        id: wait
        interval: 500
        onTriggered: win.shoot(win.contentItem, "radial.png", () => {
            hud.handle({ type: "radial", radial: { phase: "cancel", source: "right_joystick", index: 2 } })
            hud.handle({ type: "chord", chord: { button: "R4", keys: ["Meta", "Shift", "."] } })
            hud.handle({ type: "chord", chord: { button: "L5", keys: ["Ctrl", "Z"], label: "Undo" } })
            hud.handle({ type: "chord", chord: { button: "A", keys: ["Enter"] } })
            hud.handle({ type: "chord", chord: { button: "A", keys: ["Enter"] } })
            wait2.start()
        })
    }
    Timer {
        id: wait2
        interval: 500
        onTriggered: win.shoot(win.contentItem, "chords.png", () => {
            hud.toasts.clear()
            hud.handle(win.load("state.json"))
            wait3.start()
        })
    }
    Timer {
        id: wait3
        interval: 500
        onTriggered: win.shoot(win.contentItem, "sheet.png", () => {
            icon.visible = true
            win.shoot(icon, "icon.png", () => Qt.quit())
        })
    }
}
