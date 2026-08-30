import QtQuick 2.7
import QtQuick.Window 2.2
import QtMultimedia 5.8
import Ubuntu.Components 1.3

// Сканер QR: кадр с камеры сохраняется в /tmp и распознаётся демоном через zxing.
Page {
    id: page

    property string result: ""

    // Датчик установлен повёрнутым (на этом устройстве: задняя 270°, передняя 90°),
    // поэтому предпросмотр надо докрутить. uiAngle учитывает поворот самого
    // интерфейса: MainView вращается вместе с устройством.
    readonly property int uiAngle: Screen.angleBetween(Screen.orientation,
                                                       Screen.primaryOrientation)
    readonly property int sensorFix: cam.position === Camera.FrontFace
        ? (cam.orientation + uiAngle + 360) % 360
        : (360 - cam.orientation + uiAngle) % 360

    header: PageHeader {
        id: hdr
        title: root.tr("Scan QR")
        trailingActionBar.actions: [
            Action {
                iconName: "rotate-right"
                text: root.tr("Rotate preview")
                onTriggered: root.qrPreviewRotation = (root.qrPreviewRotation + 90) % 360
            },
            Action {
                iconName: "camera-flip"
                text: root.tr("Flip camera")
                visible: QtMultimedia.availableCameras.length > 1
                onTriggered: {
                    var cams = QtMultimedia.availableCameras
                    for (var i = 0; i < cams.length; i++) {
                        if (cams[i].deviceId !== cam.deviceId) {
                            cam.deviceId = cams[i].deviceId
                            break
                        }
                    }
                }
            }
        ]
    }

    onVisibleChanged: {
        if (visible) {
            page.result = ""
            msg.text = ""
            cam.start()
            shotTimer.running = true
        } else {
            shotTimer.running = false
            cam.stop()
        }
    }

    Camera {
        id: cam
        captureMode: Camera.CaptureStillImage
        focus.focusMode: Camera.FocusContinuous
        imageCapture {
            resolution: Qt.size(1280, 720)
            onImageSaved: root.api("/qrdecode", function(r, code) {
                busy.running = false
                if (r && r.text) {
                    page.result = r.text
                    shotTimer.running = false
                    cam.stop()
                    msg.text = ""
                    return
                }
                // 422 — кадр без QR: это норма, продолжаем снимать.
                if (code !== 422 && r && r.error)
                    msg.text = r.error
            }, "POST", path)
        }
    }

    VideoOutput {
        id: view
        anchors { top: hdr.bottom; left: parent.left; right: parent.right }
        height: parent.height - hdr.height - panel.height
        source: cam
        fillMode: VideoOutput.PreserveAspectCrop
        // Поправка датчика + ручная поправка пользователя (кнопка «Повернуть»).
        orientation: (page.sensorFix + root.qrPreviewRotation) % 360
        focus: page.visible
    }

    // Периодический снимок кадра: непрерывный анализ видео в QML недоступен.
    Timer {
        id: shotTimer
        interval: 1200
        repeat: true
        running: false
        onTriggered: {
            if (page.result !== "") return
            if (cam.imageCapture.ready) {
                busy.running = true
                cam.imageCapture.captureToLocation("/tmp/rocket-qr-frame.png")
            }
        }
    }

    ActivityIndicator {
        id: busy
        anchors.centerIn: view
        running: false
    }

    Column {
        id: panel
        anchors { bottom: parent.bottom; left: parent.left; right: parent.right; margins: units.gu(2) }
        spacing: units.gu(1)

        Label {
            width: parent.width
            wrapMode: Text.WrapAnywhere
            fontSize: "small"
            text: page.result !== "" ? page.result : root.tr("point the camera at a QR code")
        }
        Label {
            id: msg
            width: parent.width
            wrapMode: Text.Wrap
            fontSize: "small"
            color: UbuntuColors.red
        }
        Button {
            width: parent.width
            text: root.tr("Import as node")
            color: UbuntuColors.green
            visible: page.result !== ""
            onClicked: page.apply("node")
        }
        Button {
            width: parent.width
            text: root.tr("Add as subscription")
            visible: page.result !== "" && page.looksLikeURL(page.result)
            onClicked: page.apply("sub")
        }
        Button {
            width: parent.width
            text: root.tr("Scan again")
            visible: page.result !== ""
            onClicked: {
                page.result = ""
                cam.start()
                shotTimer.running = true
            }
        }
    }

    // looksLikeURL: подписка — это http(s)-ссылка, но http:// бывает и прокси-узлом,
    // поэтому выбор всегда за пользователем, а не за эвристикой.
    function looksLikeURL(s) {
        return s.indexOf("http://") === 0 || s.indexOf("https://") === 0
    }

    function apply(as) {
        msg.text = root.tr("working…")
        if (as === "sub") {
            root.api("/subs/add?name=" + encodeURIComponent(root.tr("QR")) +
                     "&url=" + encodeURIComponent(page.result),
                function(r, code) {
                    if (!r || r.error) { msg.text = (r && r.error) || ("HTTP " + code); return }
                    root.absorb(r); root.refresh(); stack.pop()
                }, "POST")
            return
        }
        root.api("/nodes/import?name=awg", function(r, code) {
            if (!r || r.error) { msg.text = (r && r.error) || ("HTTP " + code); return }
            root.absorb(r); root.refresh(); stack.pop()
        }, "POST", page.result)
    }
}
