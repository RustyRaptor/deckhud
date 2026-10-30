import QtQuick

// Brief "Action set › Layer" pill at the top of the screen.
Rectangle {
    id: banner

    property color accent: "#9a93e3"
    property string text
    property bool active: false
    property int timeout: 1500

    function show(newText) {
        text = newText
        active = true
        hideTimer.restart()
    }

    width: label.implicitWidth + 40
    height: label.implicitHeight + 18
    radius: height / 2
    color: Qt.rgba(0.05, 0.05, 0.08, 0.72)
    border.color: banner.accent
    border.width: 2
    opacity: active ? 1 : 0
    visible: opacity > 0
    Behavior on opacity { OpacityAnimator { duration: 150 } }

    Timer {
        id: hideTimer
        interval: banner.timeout
        onTriggered: banner.active = false
    }

    Text {
        id: label
        anchors.centerIn: parent
        text: banner.text
        color: "white"
        font.pixelSize: 20
        font.weight: Font.DemiBold
    }
}
