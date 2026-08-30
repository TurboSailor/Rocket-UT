package main

import (
	"strings"
	"testing"
)

const editConf = `# head comment
[General]
ipv6 = false

[Rule]
DOMAIN-SUFFIX,a.com,PROXY
DOMAIN-KEYWORD,tracker,REJECT
IP-CIDR,17.0.0.0/8,DIRECT
FINAL,DIRECT

[Host]
localhost = 127.0.0.1
`

func parseOrFail(t *testing.T, text string) *Conf {
	t.Helper()
	c, err := ParseConf(text)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func ruleStrings(c *Conf) []string {
	out := make([]string, 0, len(c.Rules))
	for _, r := range c.Rules {
		out = append(out, ruleText(r))
	}
	return out
}

func TestDeleteRuleLine(t *testing.T) {
	c := parseOrFail(t, editConf)
	target := c.Rules[1] // DOMAIN-KEYWORD,tracker,REJECT
	if !c.DeleteRuleLine(target.Line) {
		t.Fatal("DeleteRuleLine returned false")
	}
	after := parseOrFail(t, c.Text())
	got := ruleStrings(after)
	want := []string{
		"DOMAIN-SUFFIX,a.com,PROXY",
		"IP-CIDR,17.0.0.0/8,DIRECT",
		"FINAL,DIRECT",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("rules = %v, want %v", got, want)
	}
	// Остальной файл цел.
	for _, keep := range []string{"# head comment", "[General]", "ipv6 = false", "[Host]", "localhost = 127.0.0.1"} {
		if !strings.Contains(c.Text(), keep) {
			t.Errorf("delete damaged file: %q lost", keep)
		}
	}
}

func TestDeleteRuleLineRejectsNonRule(t *testing.T) {
	c := parseOrFail(t, editConf)
	// Строка [General] — не правило.
	if c.DeleteRuleLine(2) {
		t.Error("deleted a non-rule line")
	}
	if c.DeleteRuleLine(9999) {
		t.Error("deleted a nonexistent line")
	}
}

func TestUpdateRuleLine(t *testing.T) {
	c := parseOrFail(t, editConf)
	target := c.Rules[0]
	if !c.UpdateRuleLine(target.Line, Rule{Type: "DOMAIN", Arg: "b.org", Policy: "REJECT"}) {
		t.Fatal("UpdateRuleLine returned false")
	}
	after := parseOrFail(t, c.Text())
	if got := ruleText(after.Rules[0]); got != "DOMAIN,b.org,REJECT" {
		t.Errorf("first rule = %q", got)
	}
	if len(after.Rules) != 4 {
		t.Errorf("rule count changed: %d", len(after.Rules))
	}
}

func TestMoveRuleLine(t *testing.T) {
	t.Run("down swaps with next rule", func(t *testing.T) {
		c := parseOrFail(t, editConf)
		if _, ok := c.MoveRuleLine(c.Rules[0].Line, false); !ok {
			t.Fatal("move down failed")
		}
		got := ruleStrings(parseOrFail(t, c.Text()))
		want := []string{
			"DOMAIN-KEYWORD,tracker,REJECT",
			"DOMAIN-SUFFIX,a.com,PROXY",
			"IP-CIDR,17.0.0.0/8,DIRECT",
			"FINAL,DIRECT",
		}
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("rules = %v, want %v", got, want)
		}
	})
	t.Run("up swaps with previous rule", func(t *testing.T) {
		c := parseOrFail(t, editConf)
		if _, ok := c.MoveRuleLine(c.Rules[2].Line, true); !ok {
			t.Fatal("move up failed")
		}
		got := ruleStrings(parseOrFail(t, c.Text()))
		want := []string{
			"DOMAIN-SUFFIX,a.com,PROXY",
			"IP-CIDR,17.0.0.0/8,DIRECT",
			"DOMAIN-KEYWORD,tracker,REJECT",
			"FINAL,DIRECT",
		}
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("rules = %v, want %v", got, want)
		}
	})
	t.Run("edges refuse", func(t *testing.T) {
		c := parseOrFail(t, editConf)
		if _, ok := c.MoveRuleLine(c.Rules[0].Line, true); ok {
			t.Error("moved first rule up")
		}
		if _, ok := c.MoveRuleLine(c.Rules[len(c.Rules)-1].Line, false); ok {
			t.Error("moved last rule down")
		}
	})
}

// Порядок правил — это приоритет: перестановка должна менять исход маршрутизации.
func TestMoveRuleChangesRouting(t *testing.T) {
	text := "[Rule]\nDOMAIN-SUFFIX,x.com,DIRECT\nDOMAIN-SUFFIX,x.com,PROXY\nFINAL,DIRECT\n"
	c := parseOrFail(t, text)
	n, err := ParseURI("socks5://127.0.0.1:1080#t")
	if err != nil {
		t.Fatal(err)
	}
	firstOutbound := func(c *Conf) string {
		b, err := buildSingbox("config", &n, c, newRulesetCache())
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range b.Config["route"].(jmap)["rules"].([]jmap) {
			if ds, ok := r["domain_suffix"].([]string); ok && len(ds) > 0 && ds[0] == "x.com" {
				return r["outbound"].(string)
			}
		}
		return ""
	}
	if got := firstOutbound(c); got != tagDirect {
		t.Fatalf("before move, first match = %q, want %s", got, tagDirect)
	}
	if _, ok := c.MoveRuleLine(c.Rules[0].Line, false); !ok {
		t.Fatal("move failed")
	}
	if got := firstOutbound(parseOrFail(t, c.Text())); got != tagProxy {
		t.Errorf("after move, first match = %q, want %s", got, tagProxy)
	}
}

func TestParseRuleInput(t *testing.T) {
	ok := []struct {
		json string
		want string
	}{
		{`{"type":"DOMAIN-SUFFIX","arg":"a.com","policy":"PROXY"}`, "DOMAIN-SUFFIX,a.com,PROXY"},
		{`{"type":"domain","arg":" b.com ","policy":"direct"}`, "DOMAIN,b.com,DIRECT"},
		{`{"type":"IP-CIDR","arg":"1.2.3.4","policy":"REJECT"}`, "IP-CIDR,1.2.3.4,REJECT"},
		{`{"type":"FINAL","policy":"PROXY"}`, "FINAL,PROXY"},
	}
	for _, tc := range ok {
		t.Run(tc.json, func(t *testing.T) {
			r, err := parseRuleInput(tc.json)
			if err != nil {
				t.Fatal(err)
			}
			if got := ruleText(r); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
	bad := []string{
		`not json`,
		`{"type":"DOMAIN","arg":"","policy":"PROXY"}`,
		`{"type":"RULE-SET","arg":"http://x","policy":"PROXY"}`,
		`{"type":"IP-ASN","arg":"4134","policy":"PROXY"}`,
		`{"type":"DOMAIN","arg":"a.com","policy":"BOGUS"}`,
		`{"type":"IP-CIDR","arg":"not-an-ip","policy":"PROXY"}`,
	}
	for _, s := range bad {
		t.Run("bad:"+s, func(t *testing.T) {
			if _, err := parseRuleInput(s); err == nil {
				t.Errorf("expected error for %s", s)
			}
		})
	}
}
