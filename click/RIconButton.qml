import QtQuick 2.7
import Ubuntu.Components 1.3

// Круглая кнопка-иконка. Область нажатия всегда не меньше 4.5 gu,
// даже если сама иконка мелкая.
Item {
    id: btn

    property string name: ""
    property color tint: pal.text
    property real iconSize: units.gu(2.5)
    property bool filled: false
    property color fillColor: pal.surfaceAlt
    signal clicked()

    width: units.gu(4.5)
    height: units.gu(4.5)
    opacity: btn.enabled ? (ma.pressed ? 0.55 : 1) : 0.35

    Rectangle {
        anchors.fill: parent
        radius: width / 2
        visible: btn.filled
        color: btn.fillColor
    }

    RIcon {
        anchors.centerIn: parent
        name: btn.name
        tint: btn.tint
        size: btn.iconSize
    }

    MouseArea {
        id: ma
        anchors.fill: parent
        enabled: btn.enabled
        onClicked: btn.clicked()
    }
}
