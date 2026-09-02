import QtQuick 2.7
import Ubuntu.Components 1.3

// Плитка навигации на главном экране. Ширину задаёт вызывающий долей от сетки.
Rectangle {
    id: tile

    property string icon: ""
    property color tint: pal.accent
    property string label: ""
    property string value: ""
    signal clicked()

    height: units.gu(10)
    radius: pal.radius
    color: pal.surface
    border.width: 1
    border.color: pal.border
    opacity: ma.pressed ? 0.7 : 1

    Rectangle {
        id: badge
        width: units.gu(4.5)
        height: units.gu(4.5)
        radius: units.gu(1.2)
        anchors { left: parent.left; leftMargin: units.gu(1.5); top: parent.top; topMargin: units.gu(1.5) }
        color: Qt.rgba(tile.tint.r, tile.tint.g, tile.tint.b, 0.16)

        RIcon {
            anchors.centerIn: parent
            name: tile.icon
            tint: tile.tint
            size: units.gu(2.4)
        }
    }

    Text {
        visible: tile.value !== ""
        anchors { right: parent.right; rightMargin: units.gu(1.5); top: parent.top; topMargin: units.gu(2) }
        text: tile.value
        color: pal.dim
        font.pixelSize: pal.fsSmall
        font.weight: Font.DemiBold
    }

    Text {
        anchors {
            left: parent.left; leftMargin: units.gu(1.5)
            right: parent.right; rightMargin: units.gu(1.5)
            bottom: parent.bottom; bottomMargin: units.gu(1.5)
        }
        text: tile.label
        color: pal.text
        font.pixelSize: pal.fsBody
        font.weight: Font.DemiBold
        elide: Text.ElideRight
    }

    MouseArea {
        id: ma
        anchors.fill: parent
        onClicked: tile.clicked()
    }
}
