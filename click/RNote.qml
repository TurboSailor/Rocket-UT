import QtQuick 2.7
import Ubuntu.Components 1.3

// Полоска сообщения: ошибка, предупреждение или подсказка.
// tone: bad | warn | info
Item {
    id: note

    property string text: ""
    property string tone: "bad"
    readonly property color toneColor: tone === "warn" ? pal.warn
                                     : tone === "info" ? pal.accent : pal.bad

    visible: text !== ""
    height: visible ? body.height + units.gu(2) : 0

    Rectangle {
        anchors.fill: parent
        radius: pal.radiusSm
        color: Qt.rgba(note.toneColor.r, note.toneColor.g, note.toneColor.b, 0.12)
        border.width: 1
        border.color: Qt.rgba(note.toneColor.r, note.toneColor.g, note.toneColor.b, 0.4)
    }

    Text {
        id: body
        anchors {
            left: parent.left; leftMargin: units.gu(1.5)
            right: parent.right; rightMargin: units.gu(1.5)
            verticalCenter: parent.verticalCenter
        }
        text: note.text
        color: note.toneColor
        font.pixelSize: pal.fsSmall
        wrapMode: Text.Wrap
    }
}
