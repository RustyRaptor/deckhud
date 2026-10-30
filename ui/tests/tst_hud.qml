import QtQuick
import QtTest
import "../components"

Item {
    id: root
    width: 1280
    height: 800

    Component {
        id: hudComponent
        Hud {
            anchors.fill: parent
            accent: "#9a93e3"
        }
    }

    TestCase {
        name: "Hud"
        when: windowShown

        readonly property var items: [{ label: "Copy", keys: ["Ctrl", "C"] }, { keys: ["Ctrl", "V"] }]

        function test_radialSuppressesChords() {
            const h = createTemporaryObject(hudComponent, root)
            h.handle({ type: "radial", radial: { phase: "open", source: "right_joystick", title: "ops", items: items, index: 0 } })
            verify(h.radial.active)
            h.handle({ type: "chord", chord: { button: "A", keys: ["Enter"] } })
            compare(h.toasts.count, 0, "no toasts while a radial menu is open")
            h.handle({ type: "radial", radial: { phase: "move", source: "right_joystick", index: 1 } })
            compare(h.radial.index, 1)
            h.handle({ type: "radial", radial: { phase: "cancel", source: "right_joystick", index: 1 } })
            verify(!h.radial.active)
            h.handle({ type: "chord", chord: { button: "A", keys: ["Enter"] } })
            compare(h.toasts.count, 1)
        }

        function test_radialFire() {
            const h = createTemporaryObject(hudComponent, root)
            h.handle({ type: "radial", radial: { phase: "open", source: "right_joystick", items: items, index: 0 } })
            h.handle({ type: "radial", radial: { phase: "fire", source: "right_joystick", index: 1, keys: ["Ctrl", "V"] } })
            compare(h.radial.firedIndex, 1)
        }

        function test_stateBanner() {
            const h = createTemporaryObject(hudComponent, root)
            h.handle({ type: "state", state: { set: "Desktop", layers: ["Editing"] } })
            compare(h.banner.text, "Desktop › Editing")
            verify(h.banner.active)
            h.handle({ type: "state", state: { set: "Gamepad" } })
            compare(h.banner.text, "Gamepad")
        }

        function test_firstStateIsQuiet() {
            const h = createTemporaryObject(hudComponent, root)
            h.handle({ type: "hello", version: 1 })
            h.handle({ type: "state", state: { set: "Desktop" } })
            verify(!h.banner.active, "the greeting's state should not pop a banner")
        }

        function test_handleText() {
            const h = createTemporaryObject(hudComponent, root)
            h.handleText('{"type":"chord","chord":{"button":"R4","keys":["Meta","Shift","."]}}')
            compare(h.toasts.count, 1)
            h.handleText("not json")
            h.handle({ type: "bogus" })
            compare(h.toasts.count, 1)
        }
    
        function test_accentMessage() {
            const h = createTemporaryObject(hudComponent, root)
            h.handle({ type: "accent", color: "#f09cca" })
            tryCompare(h, "accent", Qt.color("#f09cca"), 2000)
            h.handle({ type: "accent", color: "not a colour" })
            wait(600)
            compare(h.accent, Qt.color("#f09cca"), "invalid colours are ignored")
        }
    
        function test_configSetsIconDir() {
            const h = createTemporaryObject(hudComponent, root)
            h.handle({ type: "config", app: "413080", title: "desk", iconDir: "file:///steam/icons" })
            compare(h.iconDir, "file:///steam/icons")
            compare(h.radial.iconDir, "file:///steam/icons")
        }
    }
}
