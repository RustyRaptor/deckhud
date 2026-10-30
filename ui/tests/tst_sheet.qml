import QtQuick
import QtTest
import "../components"

Item {
    id: root
    width: 1280
    height: 800

    readonly property var sheet: ({
        set: "Desktop", layers: ["Editing"],
        controls: [
            { glyph: "L1", source: "switch", mode: "switches", entries: [{ activator: "Full_Press", keys: ["Alt"] }] },
            { glyph: "L5", source: "switch", mode: "switches", changed: true,
              entries: [{ activator: "Full_Press", label: "Undo", keys: ["Ctrl", "Z"] }] },
            { glyph: "A", source: "button_diamond", mode: "four_buttons", entries: [{ activator: "Full_Press", keys: ["Enter"] }] },
            { glyph: "RS", source: "right_joystick", mode: "radial_menu",
              entries: [{ input: "touch_menu_button_1", activator: "Full_Press", keys: ["Ctrl", "V"] }] },
            { glyph: "RPad", source: "right_trackpad", mode: "absolute_mouse", entries: [] }
        ]
    })

    Component {
        id: sheetComponent
        LayerSheet {
            anchors.fill: parent
            accent: "#9a93e3"
        }
    }

    Component {
        id: hudComponent
        Hud {
            anchors.fill: parent
        }
    }

    TestCase {
        name: "LayerSheet"
        when: windowShown

        function test_splitAndTitle() {
            const s = createTemporaryObject(sheetComponent, root)
            s.show(sheet, 0)
            verify(s.active)
            compare(s.title, "Desktop › Editing")
            compare(s.leftControls.map(c => c.glyph), ["L1", "L5"])
            compare(s.rightControls.map(c => c.glyph), ["A", "RS", "RPad"])
        }

        function test_entryText_data() {
            return [
                { tag: "full", entry: { activator: "Full_Press", keys: ["F10"] }, text: "F10" },
                { tag: "long", entry: { activator: "Long_Press", keys: ["F12"] }, text: "hold  F12" },
                { tag: "double", entry: { activator: "Double_Press", keys: ["F11"] }, text: "2×  F11" },
                { tag: "label", entry: { activator: "Full_Press", label: "Undo", keys: ["Ctrl", "Z"] }, text: "Undo  (Ctrl+Z)" },
                { tag: "radial", entry: { input: "touch_menu_button_3", activator: "Full_Press", keys: ["Ctrl", "X"] }, text: "3  Ctrl+X" },
                { tag: "dpad", entry: { input: "dpad_north", activator: "Full_Press", keys: ["↑"] }, text: "▲  ↑" }
            ]
        }

        function test_entryText(data) {
            const s = createTemporaryObject(sheetComponent, root)
            compare(s.entryText(data.entry), data.text)
        }

        function test_modeText() {
            const s = createTemporaryObject(sheetComponent, root)
            compare(s.modeText("absolute_mouse"), "Mouse")
            compare(s.modeText("radial_menu"), "Radial menu")
            compare(s.modeText("some_new_mode"), "Some new mode")
        }

        function test_autoHideAndToggle() {
            const s = createTemporaryObject(sheetComponent, root)
            s.show(sheet, 100)
            verify(s.active)
            tryCompare(s, "active", false, 2000)

            s.toggle(sheet)
            verify(s.active, "toggle shows the sheet")
            wait(200)
            verify(s.active, "a toggled sheet stays until toggled again")
            s.toggle(sheet)
            verify(!s.active)
        }

        function test_hudShowsSheetOnStateAndPeek() {
            const h = createTemporaryObject(hudComponent, root)
            h.handle({ type: "state", state: { set: "Desktop", layers: ["Editing"] }, sheet: sheet })
            verify(h.sheet.active, "layer change shows the sheet")
            h.sheet.hide()
            h.handle({ type: "peek" })
            verify(h.sheet.active, "peek shows the last sheet")
            h.handle({ type: "peek" })
            verify(!h.sheet.active)
        }
    }
}
