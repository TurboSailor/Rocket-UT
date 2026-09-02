import QtQuick 2.7
import Ubuntu.Components 1.3

// Сегментированный переключатель. Ячейки — доля ширины, никакой
// фиксированной арифметики: на узком экране ничего не уезжает.
Item {
    id: seg

    property var options: []
    property int currentIndex: 0
    signal selected(int index)

    readonly property real inset: units.dp(2)
    readonly property real cellW: options.length > 0 ? (width - 2 * inset) / options.length : 0
    readonly property real cellH: height - 2 * inset

    height: units.gu(5.5)

    Rectangle {
        anchors.fill: parent
        radius: pal.radiusSm
        color: pal.surfaceAlt
        border.width: 1
        border.color: pal.border
    }

    Rectangle {
        width: seg.cellW
        height: seg.cellH
        x: seg.inset + seg.cellW * seg.currentIndex
        y: seg.inset
        radius: pal.radiusSm
        color: pal.accent
        Behavior on x { NumberAnimation { duration: 130; easing.type: Easing.OutCubic } }
    }

    Row {
        anchors.fill: parent
        anchors.margins: seg.inset

        Repeater {
            model: seg.options

            delegate: Item {
                width: seg.cellW
                height: seg.cellH

                Text {
                    anchors.centerIn: parent
                    width: parent.width - units.gu(0.5)
                    horizontalAlignment: Text.AlignHCenter
                    elide: Text.ElideRight
                    text: modelData
                    color: index === seg.currentIndex ? "#FFFFFF" : pal.dim
                    font.pixelSize: pal.fsSmall
                    font.weight: index === seg.currentIndex ? Font.DemiBold : Font.Normal
                }
                MouseArea {
                    anchors.fill: parent
                    onClicked: seg.selected(index)
                }
            }
        }
    }
}
