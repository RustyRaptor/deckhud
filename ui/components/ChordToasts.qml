pragma ComponentBehavior: Bound

import QtQuick

// Short-lived "button → keys" toasts stacked at the bottom of the screen.
Item {
    id: toasts

    property color accent: "#9a93e3"
    property int timeout: 1600
    property int maxToasts: 3
    readonly property alias model: toastModel
    readonly property int count: toastModel.count

    function push(chord) {
        const keysText = (chord.keys ?? []).join(" + ")
        const now = Date.now()
        const last = count > 0 ? toastModel.get(count - 1) : null
        if (last && last.button === chord.button && last.keysText === keysText && now < last.expires) {
            toastModel.setProperty(count - 1, "repeat", last.repeat + 1)
            toastModel.setProperty(count - 1, "expires", now + timeout)
            return
        }
        toastModel.append({ button: chord.button ?? "", keysText: keysText, label: chord.label ?? "",
                            repeat: 1, expires: now + timeout })
        while (toastModel.count > maxToasts)
            toastModel.remove(0)
    }
    function clear() { toastModel.clear() }

    ListModel { id: toastModel }

    Timer {
        interval: 50
        repeat: true
        running: toasts.count > 0
        onTriggered: {
            const now = Date.now()
            while (toastModel.count > 0 && toastModel.get(0).expires <= now)
                toastModel.remove(0)
        }
    }

    Column {
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.bottom: parent.bottom
        anchors.bottomMargin: parent.height * 0.08
        spacing: 8

        Repeater {
            model: toastModel
            delegate: Rectangle {
                id: toast
                required property string button
                required property string keysText
                required property string label
                required property int repeat
                anchors.horizontalCenter: parent.horizontalCenter
                width: content.implicitWidth + 28
                height: content.implicitHeight + 16
                radius: height / 2
                color: Qt.rgba(0.05, 0.05, 0.08, 0.72)
                border.color: Qt.alpha(toasts.accent, 0.9)
                border.width: 2

                Row {
                    id: content
                    anchors.centerIn: parent
                    spacing: 10

                    ButtonGlyph {
                        anchors.verticalCenter: parent.verticalCenter
                        button: toast.button
                        accent: toasts.accent
                    }
                    Text {
                        anchors.verticalCenter: parent.verticalCenter
                        text: "→"
                        color: Qt.alpha("white", 0.7)
                        font.pixelSize: 18
                    }
                    KeyCaps {
                        anchors.verticalCenter: parent.verticalCenter
                        keys: toast.keysText === "" ? [] : toast.keysText.split(" + ")
                        accent: toasts.accent
                    }
                    Text {
                        anchors.verticalCenter: parent.verticalCenter
                        visible: toast.label !== ""
                        text: toast.label
                        color: Qt.alpha("white", 0.8)
                        font.pixelSize: 16
                        font.italic: true
                    }
                    Text {
                        anchors.verticalCenter: parent.verticalCenter
                        visible: toast.repeat > 1
                        text: "×" + toast.repeat
                        color: toasts.accent
                        font.pixelSize: 16
                        font.bold: true
                    }
                }
            }
        }
    }
}
