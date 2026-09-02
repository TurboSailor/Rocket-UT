import QtQuick 2.7
import Ubuntu.Components 1.3

// Палитра и метрики тёмной темы. Инстанцируется в main.qml как `pal`,
// страницы и компоненты видят её через цепочку контекстов QML.
// Все размеры — units.gu/units.dp: сырые пиксели на 400+ DPI нечитаемы.
QtObject {
    readonly property color bg:         "#0B0E14"
    readonly property color surface:    "#161C27"
    readonly property color surfaceAlt: "#1E2635"
    readonly property color border:     "#27303F"
    readonly property color text:       "#EBEFF7"
    readonly property color dim:        "#8A94A8"
    readonly property color faint:      "#5C6577"
    readonly property color accent:     "#3D7BFF"
    readonly property color ok:         "#27C46B"
    readonly property color warn:       "#F5A524"
    readonly property color bad:        "#F0464B"
    readonly property color violet:     "#8B5CF6"

    readonly property real radius:   units.gu(1.5)
    readonly property real radiusSm: units.gu(1)
    readonly property real pad:      units.gu(2)
    readonly property real gap:      units.gu(1.5)
    readonly property real ctlH:     units.gu(6)
    readonly property real fieldH:   units.gu(5.5)
    readonly property real headerH:  units.gu(7)

    readonly property real fsTitle: units.dp(19)
    readonly property real fsBody:  units.dp(15)
    readonly property real fsSmall: units.dp(13)
    readonly property real fsTiny:  units.dp(11)
    readonly property real stroke:  units.dp(0.8)
}
