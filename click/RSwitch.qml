import QtQuick 2.7
import Ubuntu.Components 1.3

// Переключатель. Своя отделка, чтобы не тащить светлый UITK Switch.
Item {
    id: sw

    property bool checked: false
    signal toggled(bool value)

    width: units.gu(6)
    height: units.gu(3.2)

    Rectangle {
        anchors.fill: parent
        radius: height / 2
        color: sw.checked ? pal.accent : pal.surfaceAlt
        border.width: 1
        border.color: sw.checked ? pal.accent : pal.border

        Rectangle {
            id: knob
            width: parent.height - units.dp(4)
            height: width
            radius: width / 2
            y: units.dp(2)
            x: sw.checked ? parent.width - width - units.dp(2) : units.dp(2)
            color: sw.checked ? "#FFFFFF" : pal.dim
            Behavior on x { NumberAnimation { duration: 120; easing.type: Easing.OutCubic } }
        }
    }

    MouseArea {
        anchors.fill: parent
        onClicked: {
            sw.checked = (sw.checked === false)
            sw.toggled(sw.checked)
        }
    }
}
