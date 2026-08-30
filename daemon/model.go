package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
)

// Reality — параметры REALITY-TLS.
type Reality struct {
	PublicKey string `json:"public_key,omitempty"`
	ShortID   string `json:"short_id,omitempty"`
}

// Node — прокси-узел. Имена JSON-полей — контракт с QML, менять нельзя.
type Node struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"` // vless|vmess|socks5|http|ss|trojan|ssh|awg
	Server   string  `json:"server"`
	Port     int     `json:"port"`
	UUID     string  `json:"uuid,omitempty"`
	Password string  `json:"password,omitempty"`
	User     string  `json:"user,omitempty"`
	Method   string  `json:"method,omitempty"`
	Flow     string  `json:"flow,omitempty"`
	Network  string  `json:"network,omitempty"`
	Path     string  `json:"path,omitempty"`
	Host     string  `json:"host,omitempty"`
	TLS      bool    `json:"tls"`
	SNI      string  `json:"sni,omitempty"`
	ALPN     string  `json:"alpn,omitempty"`
	FP       string  `json:"fp,omitempty"`
	Insecure bool    `json:"insecure"`
	AlterID  int     `json:"alter_id,omitempty"`
	Reality  Reality `json:"reality,omitempty"`
	PrivKey  string  `json:"priv_key,omitempty"`
	AWGConf  string  `json:"awg_conf,omitempty"`
	SubID    string  `json:"sub_id,omitempty"`
	Latency  int     `json:"latency"`
}

// Sub — подписка.
type Sub struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	UpdatedAt int64  `json:"updated_at"`
	Count     int    `json:"count"`
	Err       string `json:"err,omitempty"`
}

// State — постоянное состояние демона.
type State struct {
	Up       bool   `json:"up"`
	Mode     string `json:"mode"` // config|proxy|direct
	NodeID   string `json:"node_id"`
	ConfName string `json:"conf_name"`
	Err      string `json:"err,omitempty"`
}

// ConnEntry — запись лога соединений.
type ConnEntry struct {
	Time   int64  `json:"time"`
	Host   string `json:"host"`
	IP     string `json:"ip"`
	Port   int    `json:"port"`
	Net    string `json:"net"`
	Rule   string `json:"rule"`
	Policy string `json:"policy"`
	Up     int64  `json:"up"`
	Down   int64  `json:"down"`
	Open   bool   `json:"open"`
}

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,32}$`)

func validName(s string) bool { return nameRe.MatchString(s) }

// id12 — стабильный короткий идентификатор от канонического ключа узла.
func id12(key string) string {
	h := sha1.Sum([]byte(key))
	return hex.EncodeToString(h[:])[:12]
}

// setID вычисляет ID узла по его сетевой идентичности (тип+адрес+креды).
func (n *Node) setID() {
	n.ID = id12(fmt.Sprintf("%s|%s|%d|%s|%s|%s|%s|%s",
		n.Type, n.Server, n.Port, n.UUID, n.Password, n.User, n.Path, n.AWGConf))
}
