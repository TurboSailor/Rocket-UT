.pragma library

// Штриховые иконки 24x24. Рисуются через Image + SVG data-URI:
// QtQuick.Shapes под MainView из Ubuntu.Components молча ничего не рисует.
var P = {
    "power":       "M12 3v9 M18.4 6.6a9 9 0 1 1-12.8 0",
    "bolt":        "M13 2 4 14h7l-1 8 9-12h-7z",
    "server":      "M4 4h16v6H4z M4 14h16v6H4z M7.5 7h.01 M7.5 17h.01",
    "globe":       "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18 M3 12h18 M12 3c2.6 3.2 2.6 14.8 0 18 M12 3c-2.6 3.2-2.6 14.8 0 18",
    "filter":      "M3 5h18l-7 8v6l-4-2v-4z",
    "clock":       "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18 M12 7.5V12l3.5 2",
    "plus":        "M12 5v14 M5 12h14",
    "download":    "M12 3v11 M7.5 10l4.5 4.5L16.5 10 M4 20h16",
    "upload":      "M12 21V10 M7.5 14.5 12 10l4.5 4.5 M4 4h16",
    "camera":      "M4 8h3.2l1.8-2h6l1.8 2H20v11H4z M12 16.6a3.3 3.3 0 1 0 0-6.6 3.3 3.3 0 0 0 0 6.6",
    "file":        "M6 3h8l4 4v14H6z M14 3v4h4",
    "folder":      "M3 6h6l2 2.5h10V20H3z",
    "chevronRight":"M9.5 5.5 16 12l-6.5 6.5",
    "chevronLeft": "M14.5 5.5 8 12l6.5 6.5",
    "chevronUp":   "M5.5 14.5 12 8l6.5 6.5",
    "chevronDown": "M5.5 9.5 12 16l6.5-6.5",
    "trash":       "M4 7h16 M9.5 7V4h5v3 M6.5 7 7.5 20h9L17.5 7",
    "edit":        "M4 20h4.2L20 8.2 15.8 4 4 15.8z M14.5 5.4 18.6 9.5",
    "refresh":     "M20 12a8 8 0 1 1-2.6-5.9 M20 4v5h-5",
    "check":       "M5 13l4.5 4.5L19 7",
    "close":       "M6 6l12 12 M18 6 6 18",
    "link":        "M9.5 14.5 14.5 9.5 M10.5 6.5 12 5a4 4 0 0 1 6 6l-1.5 1.5 M13.5 17.5 12 19a4 4 0 0 1-6-6l1.5-1.5",
    "shield":      "M12 3l8 3v6c0 5-3.4 8.2-8 9-4.6-.8-8-4-8-9V6z M9 12.2l2.2 2.3L15.5 10",
    "zoomIn":      "M11 18a7 7 0 1 0 0-14 7 7 0 0 0 0 14 M16.2 16.2 21 21 M8.2 11h5.6 M11 8.2v5.6",
    "zoomOut":     "M11 18a7 7 0 1 0 0-14 7 7 0 0 0 0 14 M16.2 16.2 21 21 M8.2 11h5.6",
    "rotate":      "M4 4v6h6 M20 20v-6h-6 M19.5 9A8 8 0 0 0 6 6.3 M4.5 15A8 8 0 0 0 18 17.7",
    "flip":        "M4 8h16v11H4z M9 4.5h6 M9.5 13h5l-1.6-1.6 M14.5 15h-5l1.6 1.6",
    "info":        "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18 M12 11v6 M12 7.6h.01",
    "warn":        "M12 4 21 20H3z M12 10v4.5 M12 17.4h.01",
    "play":        "M8 5.5 19 12 8 18.5z",
    "terminal":    "M4 5h16v14H4z M7.5 9.5 10 12l-2.5 2.5 M12.5 15h4"
}

var FILLED = {
    "dot":  "M12 7.5a4.5 4.5 0 1 0 0 9 4.5 4.5 0 0 0 0-9"
}

function rgbOf(c) {
    return "rgb(" + Math.round(c.r * 255) + "," + Math.round(c.g * 255) + "," + Math.round(c.b * 255) + ")"
}

// svg отдаёт data-URI. Цвет обязательно раскладывается в rgb() + *-opacity:
// Qt отдаёт color как #aarrggbb, чего SVG не понимает — иконка станет невидимой.
function svg(name, color, weight) {
    var head = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">'
    var col = rgbOf(color)
    var op = color.a.toFixed(3)
    var body
    if (FILLED[name] !== undefined) {
        body = '<path d="' + FILLED[name] + '" fill="' + col + '" fill-opacity="' + op + '"/>'
    } else {
        var d = P[name]
        if (d === undefined)
            return ""
        body = '<path d="' + d + '" fill="none" stroke="' + col + '" stroke-opacity="' + op
             + '" stroke-width="' + weight.toFixed(2)
             + '" stroke-linecap="round" stroke-linejoin="round"/>'
    }
    return "data:image/svg+xml;utf8," + head + body + "</svg>"
}

function has(name) {
    return P[name] !== undefined || FILLED[name] !== undefined
}
