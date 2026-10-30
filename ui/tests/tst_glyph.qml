import QtQuick
import QtTest
import "../components"

Item {
    id: root
    width: 200
    height: 200

    Component {
        id: glyphComponent
        ButtonGlyph {}
    }

    TestCase {
        name: "ButtonGlyph"
        when: windowShown

        function test_shapes_data() {
            return [
                { tag: "A", button: "A", shape: "face", text: "A" },
                { tag: "Y", button: "Y", shape: "face", text: "Y" },
                { tag: "L1", button: "L1", shape: "bumper", text: "L1" },
                { tag: "R2", button: "R2", shape: "trigger", text: "R2" },
                { tag: "L4", button: "L4", shape: "back", text: "L4" },
                { tag: "R5", button: "R5", shape: "back", text: "R5" },
                { tag: "Up", button: "Up", shape: "dpad", text: "▲" },
                { tag: "Left", button: "Left", shape: "dpad", text: "◀" },
                { tag: "L3", button: "L3", shape: "stick", text: "L3" },
                { tag: "View", button: "View", shape: "system", text: "⧉" },
                { tag: "Menu", button: "Menu", shape: "system", text: "☰" },
                { tag: "LPadClick", button: "LPadClick", shape: "pad", text: "LP" },
                { tag: "unknown", button: "Weird", shape: "system", text: "Weird" }
            ]
        }

        function test_shapes(data) {
            const g = createTemporaryObject(glyphComponent, root, { button: data.button })
            compare(g.shape, data.shape)
            compare(g.glyphText, data.text)
        }
    }
}
