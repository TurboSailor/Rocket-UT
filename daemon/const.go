package main

// Единственное место определения портов, имён интерфейсов, таблиц и марок.
// При отладке маршрутизации правится только этот файл.
const (
	APIAddr   = "127.0.0.1:8877"
	ClashAddr = "127.0.0.1:9091"

	TunName      = "rocket0"
	TunAddr4     = "172.19.0.1/30"
	TunMTU       = 1420
	TunTable     = 9877
	TunRuleIndex = 9000

	AwgName     = "awg0"
	AwgTable    = 8878
	AwgRulePrio = 8800
	AwgMark     = 0x8878
	EscapeMark  = 0x8877
	EscapePrio  = 8801
	AwgSockDir  = "/var/run/amneziawg"

	StateDir  = "/var/lib/rocketd"
	ClickRoot = "/opt/click.ubuntu.com/rocket/current"
	ClickBin  = ClickRoot + "/bin"
	LocalBin  = StateDir + "/bin"

	// Кадр с камеры, который QML отдаёт демону на распознавание.
	QRFrame = "/tmp/rocket-qr-frame.png"

	Home = "/home/phablet"

	maxBody     = 1 << 20
	maxSubBody  = 4 << 20
	connLogKeep = 5000
	connRingCap = 500

	// Теги outbound-ов в сгенерированном конфиге sing-box.
	tagDirect = "direct"
	tagProxy  = "proxy"
	tagAwg    = "awg-out"
	tagBlock  = "block"
)

var userRoots = []string{
	Home + "/Downloads",
	Home + "/Documents",
	Home + "/Pictures",
}

// Встроенный шаблон .conf: используется, когда правило добавляют, а активного .conf нет.
const defaultConf = `# Rocket default config
[General]
ipv6 = false
bypass-system = true
skip-proxy = 192.168.0.0/16,10.0.0.0/8,172.16.0.0/12,localhost,*.local

[Rule]
FINAL,DIRECT
`
