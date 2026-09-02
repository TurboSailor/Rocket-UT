import QtQuick 2.7
import Qt.labs.settings 1.0
import Ubuntu.Components 1.3
import "i18n.js" as I18n

MainView {
    id: root
    objectName: "mainView"
    applicationName: "rocket.local"
    automaticOrientation: true
    anchorToKeyboard: true
    width: units.gu(45)
    height: units.gu(75)

    // Тема UITK идёт за нашей палитрой: SuruDark/Ambiance красят то, что
    // рисует сам тулкит (TextArea, ActivityIndicator, диалоги).
    theme.name: root.darkTheme ? "Ubuntu.Components.Themes.SuruDark"
                               : "Ubuntu.Components.Themes.Ambiance"
    backgroundColor: pal.bg

    property var st: ({up: false, mode: "config"})
    property var nodes: []
    property var subs: []
    property var confs: []
    property string busy: ""
    property string startDir: "/home/phablet/Downloads"
    // pendingFile: кому отдать содержимое выбранного файла ("import" | "conf" | "qr")
    property string pendingFile: "import"
    // Ручная поправка поворота предпросмотра камеры, живёт в пределах сессии.
    property int qrPreviewRotation: 0
    // Тема: светлая по умолчанию, выбор пользователя запоминается.
    property bool darkTheme: false

    // Палитра и метрики; страницы видят её как `pal` через цепочку контекстов.
    Pal { id: pal; dark: root.darkTheme }

    Settings {
        id: uiPrefs
        category: "ui"
        property alias darkTheme: root.darkTheme
    }

    function tr(k) { return I18n.tr(k) }

    // Язык берётся из локали системы: без этого двуязычный словарь
    // всегда отдавал английский.
    Component.onCompleted: I18n.setLang(Qt.locale().name)

    // Поправка поворота кадра запоминается: подобрав её один раз,
    // пользователь не должен повторять это при каждом запуске.
    Settings {
        id: prefs
        category: "scanner"
        property alias previewRotation: root.qrPreviewRotation
    }

    function api(path, cb, method, body) {
        var x = new XMLHttpRequest()
        x.open(method || "GET", "http://127.0.0.1:8877" + path)
        x.onreadystatechange = function() {
            if (x.readyState === XMLHttpRequest.DONE) {
                var r = null
                try { r = JSON.parse(x.responseText) } catch(e) {}
                if (cb) cb(r, x.status)
            }
        }
        x.send(body || null)
    }

    // absorb обновляет общее состояние из любого ответа демона.
    function absorb(r) {
        if (!r) { root.st = {up: false, error: tr("daemon unavailable")}; return }
        // Демон отдаёт null для пустых списков — приводим к массиву,
        // иначе .length в делегатах падает.
        if (r.nodes !== undefined) root.nodes = r.nodes || []
        if (r.subs !== undefined) root.subs = r.subs || []
        if (r.confs !== undefined) root.confs = r.confs || []
        if (r.up !== undefined) root.st = r
    }

    function refresh() {
        api("/status", function(r) { absorb(r) })
        api("/nodes", function(r) { if (r && r.nodes) root.nodes = r.nodes })
    }

    function fmtBytes(b) {
        if (b === undefined || b === null) return "—"
        if (b >= 1073741824) return (b/1073741824).toFixed(1) + " GB"
        if (b >= 1048576)    return (b/1048576).toFixed(1) + " MB"
        if (b >= 1024)       return (b/1024).toFixed(0) + " KB"
        return b + " B"
    }

    function fmtAgo(ts) {
        if (!ts) return tr("never")
        var d = Math.max(0, Math.floor(Date.now()/1000) - ts)
        if (d < 60) return d + "s"
        if (d < 3600) return Math.floor(d/60) + "m"
        return Math.floor(d/3600) + "h"
    }

    function fmtTime(ts) {
        var d = new Date(ts * 1000)
        return Qt.formatTime(d, "hh:mm:ss")
    }

    function nodeLabel(n) {
        return n.name + "  ·  " + n.type + (n.server ? "  " + n.server : "")
    }

    // Цвет плашки протокола: узлы различаются на глаз без чтения подписи.
    function typeColor(t) {
        if (t === "vless" || t === "vmess") return pal.accent
        if (t === "ss" || t === "trojan") return pal.violet
        if (t === "awg") return pal.ok
        if (t === "ssh") return pal.warn
        return pal.dim
    }

    function joinPath(dir, name) {
        if (!dir || dir === "/") return "/" + name
        return dir + "/" + name
    }

    Timer {
        interval: 2000
        running: true
        repeat: true
        onTriggered: api("/status", function(r) { absorb(r) })
    }

    PageStack {
        id: stack
        Component.onCompleted: { stack.push(home); root.refresh() }
        HomePage    { id: home;      visible: false }
        NodesPage   { id: nodesPage; visible: false }
        NodeEditPage { id: nodeEdit; visible: false }
        SubsPage    { id: subsPage;  visible: false }
        ConfsPage   { id: confsPage; visible: false }
        RulesPage   { id: rulesPage; visible: false }
        LogPage     { id: logPage;   visible: false }
        ImportPage  { id: importPage; visible: false }
        ScanPage    { id: scanPage;  visible: false }
        FilePage    { id: filePage;  visible: false }
        ConfEditPage { id: confEdit; visible: false }
    }
}
