pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Shapes

// Radial menu mirror: item 0 centred at the top, clockwise.
Item {
    id: radial

    property color accent: "#9a93e3"
    property string iconDir: ""
    property var items: []
    property string title
    property int index: -1
    property int firedIndex: -1
    property bool active: false

    readonly property int count: items.length
    readonly property real wedgeSweep: count > 0 ? 360 / count : 360
    readonly property real outerRadius: Math.min(width, height) * 0.3
    readonly property real innerRadius: outerRadius * 0.4
    readonly property var current: index >= 0 && index < count ? items[index] : null

    function open(newItems, newTitle, newIndex) {
        hideTimer.stop()
        items = newItems
        title = newTitle
        firedIndex = -1
        index = newIndex
        active = true
    }
    function select(i) { index = i }
    function fire(i) {
        index = i
        firedIndex = i
        hideTimer.restart()
    }
    function cancel() {
        hideTimer.stop()
        firedIndex = -1
        active = false
    }
    function wedgeStart(i) { return -90 - wedgeSweep / 2 + i * wedgeSweep }
    function labelPoint(i) {
        const a = (wedgeStart(i) + wedgeSweep / 2) * Math.PI / 180
        const r = (outerRadius + innerRadius) / 2
        return Qt.point(width / 2 + r * Math.cos(a), height / 2 + r * Math.sin(a))
    }
    function itemText(item) {
        if (!item) return ""
        return item.label || (item.keys ?? []).join("+")
    }

    opacity: active ? 1 : 0
    visible: opacity > 0
    Behavior on opacity { OpacityAnimator { duration: 120 } }

    Timer {
        id: hideTimer
        interval: 350
        onTriggered: radial.active = false
    }

    Repeater {
        model: radial.count
        delegate: Shape {
            id: wedge
            required property int index
            readonly property bool highlighted: index === radial.index
            readonly property bool fired: index === radial.firedIndex
            anchors.fill: parent
            preferredRendererType: Shape.CurveRenderer

            ShapePath {
                fillColor: wedge.fired ? Qt.alpha(radial.accent, 0.9)
                         : wedge.highlighted ? Qt.alpha(radial.accent, 0.6)
                         : Qt.rgba(0.06, 0.06, 0.09, 0.62)
                strokeColor: Qt.alpha(radial.accent, 0.75)
                strokeWidth: 2
                PathAngleArc {
                    centerX: radial.width / 2
                    centerY: radial.height / 2
                    radiusX: radial.outerRadius
                    radiusY: radial.outerRadius
                    startAngle: radial.wedgeStart(wedge.index)
                    sweepAngle: radial.wedgeSweep
                }
                PathAngleArc {
                    moveToStart: false
                    centerX: radial.width / 2
                    centerY: radial.height / 2
                    radiusX: radial.innerRadius
                    radiusY: radial.innerRadius
                    startAngle: radial.wedgeStart(wedge.index) + radial.wedgeSweep
                    sweepAngle: -radial.wedgeSweep
                }
            }
        }
    }

    Repeater {
        model: radial.items
        delegate: Column {
            id: entry
            required property var modelData
            required property int index
            readonly property point at: radial.labelPoint(index)
            x: at.x - width / 2
            y: at.y - height / 2
            spacing: 2

            Image {
                anchors.horizontalCenter: parent.horizontalCenter
                visible: status === Image.Ready
                source: entry.modelData.icon && radial.iconDir ? radial.iconDir + "/" + entry.modelData.icon : ""
                sourceSize: Qt.size(radial.outerRadius * 0.22, radial.outerRadius * 0.22)
                asynchronous: true
            }
            Text {
                anchors.horizontalCenter: parent.horizontalCenter
                width: Math.min(implicitWidth, radial.outerRadius * 0.6)
                horizontalAlignment: Text.AlignHCenter
                elide: Text.ElideRight
                text: radial.itemText(entry.modelData)
                color: "white"
                font.pixelSize: Math.max(12, radial.outerRadius * 0.085)
                font.weight: entry.index === radial.index ? Font.Bold : Font.Normal
                style: Text.Outline
                styleColor: Qt.rgba(0, 0, 0, 0.5)
            }
        }
    }

    Rectangle {
        anchors.centerIn: parent
        width: radial.innerRadius * 2 - 8
        height: width
        radius: width / 2
        color: Qt.rgba(0.04, 0.04, 0.06, 0.78)
        border.color: Qt.alpha(radial.accent, 0.5)
        border.width: 1

        Column {
            anchors.centerIn: parent
            width: parent.width * 0.8
            spacing: 4

            Text {
                width: parent.width
                visible: radial.title !== ""
                horizontalAlignment: Text.AlignHCenter
                elide: Text.ElideRight
                text: radial.title
                color: Qt.alpha("white", 0.65)
                font.pixelSize: Math.max(10, radial.innerRadius * 0.16)
            }
            KeyCaps {
                anchors.horizontalCenter: parent.horizontalCenter
                keys: radial.current?.keys ?? []
                accent: radial.accent
                capHeight: Math.max(18, radial.innerRadius * 0.24)
                spacing: 3
            }
        }
    }
}
