import QtQuick 2.7
import Ubuntu.Components 1.3
import "icons.js" as Icons

// Штриховая иконка. Толщина задаётся в экранных пикселях и пересчитывается
// в единицы viewBox (24), иначе линия «плывёт» с размером иконки.
Item {
    id: ic

    property string name: ""
    property color tint: pal.text
    property real size: units.gu(2.5)
    property real weight: units.dp(0.9)

    width: size
    height: size

    Image {
        anchors.fill: parent
        smooth: true
        cache: true
        sourceSize.width: Math.max(1, Math.round(ic.size))
        sourceSize.height: Math.max(1, Math.round(ic.size))
        source: ic.name === "" ? "" : Icons.svg(ic.name, ic.tint,
                                                ic.weight * 24 / Math.max(1, ic.size))
    }
}
