import QtQuick
import QtWebSockets
import org.kde.layershell as LayerShell
import "components"

// Full-screen, click-through overlay. Usage: qml main.qml -- ws://127.0.0.1:PORT/TOKEN
Window {
    id: window

    readonly property string daemonUrl: Qt.application.arguments.find(a => a.startsWith("ws://")) ?? ""

    visible: true
    color: "transparent"
    flags: Qt.FramelessWindowHint | Qt.WindowTransparentForInput | Qt.WindowDoesNotAcceptFocus
    title: "deckhud"

    LayerShell.Window.scope: "deckhud"
    LayerShell.Window.layer: LayerShell.Window.LayerOverlay
    LayerShell.Window.anchors: LayerShell.Window.AnchorTop | LayerShell.Window.AnchorBottom
                             | LayerShell.Window.AnchorLeft | LayerShell.Window.AnchorRight
    LayerShell.Window.keyboardInteractivity: LayerShell.Window.KeyboardInteractivityNone
    LayerShell.Window.exclusionZone: -1
    LayerShell.Window.activateOnShow: false

    Hud {
        id: hud
        anchors.fill: parent
    }

    WebSocket {
        id: socket
        url: window.daemonUrl
        active: window.daemonUrl !== ""
        onTextMessageReceived: message => hud.handleText(message)
        onStatusChanged: {
            if (socket.status === WebSocket.Error)
                console.warn("deckhud: socket error:", socket.errorString)
            if (socket.status === WebSocket.Error || socket.status === WebSocket.Closed)
                reconnect.restart()
        }
    }

    Timer {
        id: reconnect
        interval: 1000
        onTriggered: {
            socket.active = false
            socket.active = true
        }
    }
}
