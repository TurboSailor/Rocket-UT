import QtQuick 2.7
import Ubuntu.Components 1.3

// Однострочное поле ввода в тёмной отделке: TextInput ядра QtQuick,
// потому что UITK TextField тянет собственную светлую тему.
FocusScope {
    id: f

    property alias text: input.text
    property alias placeholderText: ph.text
    property alias inputMethodHints: input.inputMethodHints
    property alias echoMode: input.echoMode
    property alias maximumLength: input.maximumLength
    property alias readOnly: input.readOnly
    property alias horizontalAlignment: input.horizontalAlignment
    signal accepted()

    height: pal.fieldH

    Rectangle {
        anchors.fill: parent
        radius: pal.radiusSm
        color: pal.surfaceAlt
        border.width: 1
        border.color: input.activeFocus ? pal.accent : pal.border

        TextInput {
            id: input
            anchors {
                left: parent.left; leftMargin: units.gu(1.5)
                right: parent.right; rightMargin: units.gu(1.5)
                verticalCenter: parent.verticalCenter
            }
            clip: true
            color: pal.text
            selectionColor: pal.accent
            selectedTextColor: "#FFFFFF"
            selectByMouse: true
            font.pixelSize: pal.fsBody
            onAccepted: f.accepted()
        }

        Text {
            id: ph
            anchors { left: input.left; right: input.right; verticalCenter: input.verticalCenter }
            visible: input.text === ""
            color: pal.faint
            font.pixelSize: pal.fsBody
            elide: Text.ElideRight
        }
    }
}
