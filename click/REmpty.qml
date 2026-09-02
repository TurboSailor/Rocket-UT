import QtQuick 2.7
import Ubuntu.Components 1.3

// Пустое состояние списка: значок и одна строка пояснения.
Item {
    id: empty

    property string icon: "info"
    property string text: ""

    width: units.gu(30)
    height: units.gu(14)

    Column {
        anchors.centerIn: parent
        spacing: units.gu(1.5)

        RIcon {
            anchors.horizontalCenter: parent.horizontalCenter
            name: empty.icon
            tint: pal.faint
            size: units.gu(5)
        }
        Text {
            anchors.horizontalCenter: parent.horizontalCenter
            width: empty.width
            horizontalAlignment: Text.AlignHCenter
            wrapMode: Text.Wrap
            text: empty.text
            color: pal.dim
            font.pixelSize: pal.fsBody
        }
    }
}
