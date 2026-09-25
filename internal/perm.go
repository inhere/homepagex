package internal

import (
	"fmt"
	"strings"
)

// Perm 权限等级：rw 读写 / ro 只读 / no 拒绝
type Perm string

const (
	// PermRW 读写权限
	PermRW Perm = "rw"
	// PermRO 只读权限
	PermRO Perm = "ro"
	// PermNO 拒绝访问
	PermNO Perm = "no"
)

// ValidPerm 是否是受支持的权限值
func ValidPerm(p string) bool {
	switch Perm(p) {
	case PermRW, PermRO, PermNO:
		return true
	}
	return false
}

// permRank 权限的严格程度，用于「同等具体度时的优先级」比较。
// no 最优先 —— 同级冲突时取更严格的，保证默认安全。
func permRank(p Perm) int {
	switch p {
	case PermNO:
		return 3
	case PermRW:
		return 2
	case PermRO:
		return 1
	default:
		return 0
	}
}

// ruleKind 规则的匹配方式
type ruleKind int

const (
	// ruleAll 匹配所有路径（*、/*、/**）
	ruleAll ruleKind = iota
	// ruleSubtree 子树匹配：/a 命中 /a 与 /a/...，但不命中 /ab
	ruleSubtree
	// rulePrefix 前缀匹配（兼容旧写法 /a*，会命中 /ab）
	rulePrefix
)

// pathRule 归一化后的单条路径规则
type pathRule struct {
	// Pattern 归一化后的匹配前缀
	Pattern string
	// Perm 该规则授予或拒绝的权限
	Perm Perm
	// Kind 匹配方式
	Kind ruleKind
	// Raw 原始配置文本，仅用于展示与排障
	Raw string
}

// match 判断规则是否命中该请求路径
func (r *pathRule) match(reqPath string) bool {
	switch r.Kind {
	case ruleAll:
		return true
	case rulePrefix:
		return strings.HasPrefix(reqPath, r.Pattern)
	default: // ruleSubtree
		return reqPath == r.Pattern || strings.HasPrefix(reqPath, r.Pattern+"/")
	}
}

// specificity 规则具体度：pattern 越长越具体，用于「更具体的规则优先」
func (r *pathRule) specificity() int {
	if r.Kind == ruleAll {
		return 0
	}
	return len(r.Pattern)
}

// String 返回便于展示的 "path:perm" 文本
func (r *pathRule) String() string {
	return r.displayPath() + ":" + string(r.Perm)
}

// displayPath 返回不含权限后缀的路径文本，用于前端展示
func (r *pathRule) displayPath() string {
	switch r.Kind {
	case ruleAll:
		return "/*"
	case rulePrefix:
		return r.Pattern + "*"
	default:
		return r.Pattern
	}
}

// normalizeReqPath 归一化请求路径，保证以 / 开头
func normalizeReqPath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}

// normalizePattern 把配置里的路径写法归一化成 (匹配前缀, 匹配方式)
//
//	*、/*、/**   → 匹配一切
//	/a/**        → 子树匹配
//	/a*          → 前缀匹配（兼容旧写法，会命中 /ab）
//	/a           → 子树匹配（/a 与 /a/...）
func normalizePattern(raw string) (string, ruleKind) {
	p := strings.TrimSpace(raw)

	switch p {
	case "", "*", "/*", "/**":
		return "/", ruleAll
	}

	if strings.HasSuffix(p, "/**") {
		base := strings.TrimSuffix(p, "/**")
		if base == "" {
			return "/", ruleAll
		}
		if !strings.HasPrefix(base, "/") {
			base = "/" + base
		}
		return base, ruleSubtree
	}

	if strings.HasSuffix(p, "*") {
		base := strings.TrimSuffix(p, "*")
		if base == "" {
			return "/", ruleAll
		}
		if !strings.HasPrefix(base, "/") {
			base = "/" + base
		}
		// /a/ 与 /a 等价，统一去掉结尾的 /
		return strings.TrimSuffix(base, "/"), rulePrefix
	}

	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p, ruleSubtree
}

// parseRuleToken 解析单条规则文本 "path[:perm]"。
//
// 权限后缀用最后一个 ':' 切分（修复 "/a:no" 曾被解析成 pattern=/a, perm="no:ro" 的问题）；
// 无法识别的后缀直接报错，避免静默降级成错误语义。
func parseRuleToken(token string) (*pathRule, error) {
	raw := strings.TrimSpace(token)
	if raw == "" {
		return nil, nil
	}

	pattern := raw
	perm := PermRO

	if idx := strings.LastIndex(raw, ":"); idx >= 0 {
		suffix := raw[idx+1:]
		if ValidPerm(suffix) {
			perm = Perm(suffix)
			pattern = raw[:idx]
		} else if suffix != "" && !strings.Contains(suffix, "/") {
			return nil, fmt.Errorf("unknown permission suffix %q in rule %q (expect rw|ro|no)", suffix, raw)
		}
	}

	norm, kind := normalizePattern(pattern)
	return &pathRule{Pattern: norm, Perm: perm, Kind: kind, Raw: raw}, nil
}

// Access 一次鉴权的结果
type Access struct {
	// Perm 最终权限
	Perm Perm
	// Allowed 是否允许该请求（写操作会额外要求 rw）
	Allowed bool
	// CanWrite 是否允许写
	CanWrite bool
	// Authed 是否凭已登录身份通过
	Authed bool
	// MatchedBy 命中的规则，便于日志与排障
	MatchedBy string
}

// bestMatch 从规则集中选出命中的「最具体」规则；
// 具体度相同时取更严格的权限（no > rw > ro）。
func bestMatch(rules []*pathRule, reqPath string) (*pathRule, bool) {
	var best *pathRule

	for _, r := range rules {
		if !r.match(reqPath) {
			continue
		}
		if best == nil {
			best = r
			continue
		}

		switch {
		case r.specificity() > best.specificity():
			best = r
		case r.specificity() == best.specificity() && permRank(r.Perm) > permRank(best.Perm):
			best = r
		}
	}

	return best, best != nil
}

// Resolve 统一的鉴权入口。
//
// 优先级（从高到低）：
//  1. deny 硬拒绝 —— 任何人都不能访问
//  2. 写操作必须已登录（避免匿名 *:rw 时各处结论不一致）
//  3. 已登录：用户规则命中则用之（含显式 :no）
//  4. 已登录：用户规则未命中 → 回退匿名基线（`!` 认证墙已越过）
//  5. 匿名：命中 `!` 认证墙 → 拒绝（401，前端弹登录框）
//  6. 都没有命中 → 拒绝（fail closed）
func (c *Config) Resolve(username, reqPath string, isWrite bool) Access {
	reqPath = normalizeReqPath(reqPath)

	// 1) 硬拒绝
	if r, ok := bestMatch(c.hardDenyRules, reqPath); ok {
		return Access{Perm: PermNO, MatchedBy: "deny " + r.Raw}
	}

	// 2) 写操作必须登录
	if isWrite && username == "" {
		return Access{Perm: PermNO, MatchedBy: "<write-requires-login>"}
	}

	// 3) 匿名基线（只含允许规则，不含 `!` 认证墙）
	var base Perm
	matched := false
	if r, ok := bestMatch(c.guestRules, reqPath); ok {
		base, matched = r.Perm, true
	}

	if username == "" {
		// 5) 匿名：认证墙
		if r, ok := bestMatch(c.guestDenyRules, reqPath); ok {
			return Access{Perm: PermNO, MatchedBy: "guest-deny " + r.Raw}
		}
	} else if auth, ok := c.parsedAuths[username]; ok {
		// 3) 用户自身规则优先
		if r, hit := bestMatch(auth.Rules, reqPath); hit {
			base, matched = r.Perm, true
		}
	}

	if !matched || base == PermNO {
		return Access{Perm: PermNO, MatchedBy: "<default-deny>"}
	}

	acc := Access{Perm: base, Allowed: true, CanWrite: base == PermRW, Authed: username != ""}
	if isWrite && !acc.CanWrite {
		acc.Allowed = false
	}
	return acc
}

// IsNeedAuth 匿名访问该路径是否需要登录/是否被拒绝。
// 保留这个名字是为了兼容既有调用与测试，语义即 Resolve 的匿名结果取反。
func (c *Config) IsNeedAuth(reqPath string, isWrite bool) bool {
	return !c.Resolve("", reqPath, isWrite).Allowed
}

// FilterNavsByPermission 过滤出当前身份有权访问的导航项，与 API 鉴权使用同一套 Resolve。
//
// 说明：URL 不以 "/" 开头的导航项（例如外链）不属于受权限管辖的页面，一律保留。
func (c *Config) FilterNavsByPermission(navs []NavItem, username string) []NavItem {
	var filtered []NavItem

	for _, nav := range navs {
		if nav.URL == "" || !strings.HasPrefix(nav.URL, "/") {
			filtered = append(filtered, nav)
			continue
		}
		if c.Resolve(username, nav.URL, false).Allowed {
			filtered = append(filtered, nav)
		}
	}

	return filtered
}
