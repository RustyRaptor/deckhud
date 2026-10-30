pragma ComponentBehavior: Bound

import QtQuick

// Renders ["Ctrl", "V"] as key caps joined by "+".
Row {
    id: caps

    property var keys: []
    property color accent: "#9a93e3"
    property real capHeight: 34

    spacing: 6

    Repeater {
        model: caps.keys
        delegate: Row {
            id: cap
            required property string modelData
            required property int index
            spacing: caps.spacing

            Text {
                visible: cap.index > 0
                anchors.verticalCenter: parent.verticalCenter
                text: "+"
                color: Qt.alpha("white", 0.7)
                font.pixelSize: caps.capHeight * 0.45
            }
            Rectangle {
                height: caps.capHeight
                width: Math.max(height, keyText.implicitWidth + 18)
                radius: 6
                color: Qt.rgba(0, 0, 0, 0.35)
                border.color: Qt.alpha(caps.accent, 0.85)
                border.width: 1.5

                Text {
                    id: keyText
                    anchors.centerIn: parent
                    text: cap.modelData
                    color: "white"
                    font.pixelSize: caps.capHeight * 0.45
                    font.weight: Font.DemiBold
                }
            }
        }
    }
}
