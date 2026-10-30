import QtQuick
import QtTest
import "../components"

Item {
    id: root
    width: 800
    height: 600

    Component {
        id: radialComponent
        RadialHud {
            anchors.fill: parent
            accent: "#9a93e3"
        }
    }

    TestCase {
        name: "RadialHud"
        when: windowShown

        readonly property var fourItems: [
            { label: "Copy", keys: ["Ctrl", "C"] },
            { keys: ["Ctrl", "V"] },
            { label: "Undo", keys: ["Ctrl", "Z"] },
            { keys: ["Del"] }
        ]

        function test_wedgeGeometry() {
            const r = createTemporaryObject(radialComponent, root)
            r.open(fourItems, "", 0)
            compare(r.wedgeSweep, 90)
            // PathAngleArc: 0° is 3 o'clock, clockwise. Item 0 is centred at the top.
            compare(r.wedgeStart(0), -135)
            compare(r.wedgeStart(1), -45)
            const top = r.labelPoint(0)
            fuzzyCompare(top.x, r.width / 2, 0.5)
            verify(top.y < r.height / 2)
            verify(r.labelPoint(1).x > r.width / 2, "item 2 sits clockwise, to the right")
        }

        function test_lifecycle() {
            const r = createTemporaryObject(radialComponent, root)
            verify(!r.active)
            r.open(fourItems, "text ops", 1)
            verify(r.active)
            compare(r.index, 1)
            compare(r.title, "text ops")
            compare(r.count, 4)
            r.select(2)
            compare(r.index, 2)
            r.fire(2)
            compare(r.firedIndex, 2)
            tryCompare(r, "active", false, 2000)

            r.open(fourItems, "", 0)
            r.cancel()
            verify(!r.active)
            compare(r.firedIndex, -1)
        }

        function test_itemText() {
            const r = createTemporaryObject(radialComponent, root)
            compare(r.itemText({ label: "Copy", keys: ["Ctrl", "C"] }), "Copy")
            compare(r.itemText({ keys: ["Ctrl", "V"] }), "Ctrl+V")
            compare(r.itemText({}), "")
        }
    }
}
