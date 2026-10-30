import QtQuick
import QtTest
import "../components"

Item {
    id: root
    width: 800
    height: 600

    Component {
        id: toastsComponent
        ChordToasts {
            anchors.fill: parent
            accent: "#9a93e3"
        }
    }

    TestCase {
        name: "ChordToasts"
        when: windowShown

        function chord(button, keys) {
            return { button: button, keys: keys, activator: "Full_Press" }
        }

        function test_pushAndCoalesce() {
            const t = createTemporaryObject(toastsComponent, root)
            compare(t.count, 0)
            t.push(chord("R4", ["Ctrl", "V"]))
            compare(t.count, 1)
            t.push(chord("R4", ["Ctrl", "V"]))
            compare(t.count, 1, "a repeat of the newest toast coalesces")
            compare(t.model.get(0).repeat, 2)
            t.push(chord("A", ["Enter"]))
            compare(t.count, 2)
            compare(t.model.get(1).keysText, "Enter")
        }

        function test_limit() {
            const t = createTemporaryObject(toastsComponent, root, { maxToasts: 3 })
            for (const k of ["1", "2", "3", "4", "5"])
                t.push(chord("A", [k]))
            compare(t.count, 3)
            compare(t.model.get(0).keysText, "3", "oldest toasts are dropped first")
        }

        function test_expiry() {
            const t = createTemporaryObject(toastsComponent, root, { timeout: 150 })
            t.push(chord("B", ["Esc"]))
            compare(t.count, 1)
            tryCompare(t, "count", 0, 2000)
        }

        function test_labelAndClear() {
            const t = createTemporaryObject(toastsComponent, root)
            t.push({ button: "L5", keys: ["Ctrl", "Z"], label: "Undo" })
            compare(t.model.get(0).label, "Undo")
            t.clear()
            compare(t.count, 0)
        }
    }
}
