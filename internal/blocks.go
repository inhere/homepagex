package internal

import (
	"errors"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

// BlockKind 块的类型
type BlockKind string

const (
	// BlockService 一个服务分组（services 数组的元素）
	BlockService BlockKind = "service"
	// BlockItem 一个链接项（某个分组的 items 数组的元素）
	BlockItem BlockKind = "item"
)

// BlockAction 对块的操作
type BlockAction string

const (
	BlockUpdate BlockAction = "update"
	BlockInsert BlockAction = "insert"
	BlockDelete BlockAction = "delete"
)

// BlockInfo 一个可单独编辑的块。
//
// 设计要点：只改一块时不能「解析成对象 → 整个文件重新 dump」，
// 那会把用户的注释、空行、字段顺序全部洗掉。
// 因此这里返回源码行区间，由 applyPageBlock 做文本级替换。
type BlockInfo struct {
	Kind BlockKind `json:"kind"`
	// Index services 内的下标；对 item 无意义
	Index int `json:"index"`
	// ServiceIndex item 所属分组的下标；对 service 无意义
	ServiceIndex int `json:"service_index,omitempty"`
	// Name / Icon 分组的名称与图标，便于前端直接展示
	Name string `json:"name,omitempty"`
	Icon string `json:"icon,omitempty"`
	// Value item 的解析结果，供表单编辑
	Value *Item `json:"value,omitempty"`
	// LineRange 该块在源码中的行区间 [start, end]，均为 1-based 闭区间
	LineRange [2]int `json:"line_range"`
	// YAML 该块的原始文本，供源码模式编辑
	YAML string `json:"yaml,omitempty"`

	// indent 该块起始行的缩进（列数-1），仅内部使用
	indent int
}

// BlockRequest 一次块编辑请求
type BlockRequest struct {
	Kind         BlockKind   `json:"kind"`
	Action       BlockAction `json:"action"`
	ServiceIndex int         `json:"service_index"`
	Index        int         `json:"index"`
	// YAML 该块的新内容（update / insert 时需要）
	YAML string `json:"yaml"`
}

// maxNodeLine 返回节点子树中出现的最大行号
func maxNodeLine(n ast.Node) int {
	if n == nil {
		return 0
	}

	line := 0
	if t := n.GetToken(); t != nil && t.Position != nil {
		line = t.Position.Line
	}

	pick := func(l int) {
		if l > line {
			line = l
		}
	}

	switch v := n.(type) {
	case *ast.MappingNode:
		for _, mv := range v.Values {
			pick(maxNodeLine(mv.Key))
			pick(maxNodeLine(mv.Value))
		}
	case *ast.SequenceNode:
		for _, el := range v.Values {
			pick(maxNodeLine(el))
		}
	}
	return line
}

// entryStartLine 取序列第 i 个元素的起始行（即 "- " 所在行）
func entryStartLine(seq *ast.SequenceNode, i int) int {
	if i < len(seq.Entries) && seq.Entries[i].Start != nil {
		return seq.Entries[i].Start.Position.Line
	}
	if t := seq.Values[i].GetToken(); t != nil && t.Position != nil {
		return t.Position.Line
	}
	return 0
}

// entryStartCol 取序列第 i 个元素起始行的缩进
func entryStartCol(seq *ast.SequenceNode, i int) int {
	if i < len(seq.Entries) && seq.Entries[i].Start != nil {
		return seq.Entries[i].Start.Position.Column - 1
	}
	if t := seq.Values[i].GetToken(); t != nil && t.Position != nil {
		return t.Position.Column - 1
	}
	return 0
}

// mappingField 在映射节点中查找指定 key
func mappingField(m *ast.MappingNode, name string) *ast.MappingValueNode {
	for _, mv := range m.Values {
		if key, ok := mv.Key.(*ast.StringNode); ok && key.Value == name {
			return mv
		}
	}
	return nil
}

// mappingSeqField 取映射节点中指定 key 的序列值，不存在时返回 nil
func mappingSeqField(m *ast.MappingNode, name string) *ast.SequenceNode {
	mv := mappingField(m, name)
	if mv == nil {
		return nil
	}
	if seq, ok := mv.Value.(*ast.SequenceNode); ok {
		return seq
	}
	return nil
}

// parsePageAST 解析页面 YAML 为 AST
func parsePageAST(content []byte) (*ast.MappingNode, error) {
	f, err := parser.ParseBytes(content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse page yaml: %w", err)
	}
	if len(f.Docs) == 0 {
		return nil, errors.New("page yaml is empty")
	}
	body, ok := f.Docs[0].Body.(*ast.MappingNode)
	if !ok {
		return nil, errors.New("page yaml root must be a mapping")
	}
	return body, nil
}

// servicesSeq 取页面根部的 services 序列
func servicesSeq(body *ast.MappingNode) *ast.SequenceNode {
	return mappingSeqField(body, "services")
}

// listPageBlocks 列出页面里所有可单独编辑的块
func listPageBlocks(content []byte) ([]BlockInfo, error) {
	body, err := parsePageAST(content)
	if err != nil {
		return nil, err
	}

	services := servicesSeq(body)
	if services == nil {
		return nil, errors.New("page yaml has no services list")
	}

	lines := splitLines(content)

	var blocks []BlockInfo
	for si, el := range services.Values {
		svcNode, ok := el.(*ast.MappingNode)
		if !ok {
			continue
		}

		start := entryStartLine(services, si)
		end := maxNodeLine(el)
		if start == 0 || end == 0 {
			continue
		}

		svc := BlockInfo{
			Kind:      BlockService,
			Index:     si,
			LineRange: [2]int{start, end},
			indent:    entryStartCol(services, si),
			YAML:      joinLines(lines, start, end),
		}
		if key := mappingField(svcNode, "name"); key != nil {
			if v, ok := key.Value.(*ast.StringNode); ok {
				svc.Name = v.Value
			}
		}
		if key := mappingField(svcNode, "icon"); key != nil {
			if v, ok := key.Value.(*ast.StringNode); ok {
				svc.Icon = v.Value
			}
		}
		blocks = append(blocks, svc)

		items := mappingSeqField(svcNode, "items")
		if items == nil {
			continue
		}
		for ii, itemEl := range items.Values {
			iStart := entryStartLine(items, ii)
			iEnd := maxNodeLine(itemEl)
			if iStart == 0 || iEnd == 0 {
				continue
			}

			itemText := joinLines(lines, iStart, iEnd)
			info := BlockInfo{
				Kind:         BlockItem,
				Index:        ii,
				ServiceIndex: si,
				LineRange:    [2]int{iStart, iEnd},
				indent:       entryStartCol(items, ii),
				YAML:         itemText,
			}
			// 解析出对象供表单编辑
			var parsed []Item
			if err := yaml.Unmarshal([]byte(itemText), &parsed); err == nil && len(parsed) == 1 {
				item := parsed[0]
				info.Value = &item
			}
			blocks = append(blocks, info)
		}
	}

	return blocks, nil
}

// splitLines 按行切分，保留原始行内容（不含换行符）
func splitLines(content []byte) []string {
	return strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
}

// joinLines 取 [start, end] 行区间（1-based 闭区间）的文本
func joinLines(lines []string, start, end int) string {
	if start < 1 || end > len(lines) || start > end {
		return ""
	}
	return strings.Join(lines[start-1:end], "\n")
}

// normalizeBlockYAML 把客户端给的块文本规范成可直接插入指定缩进位置的形式：
// 统一缩进、补上序列标记 "- "、去掉多余的尾随空行。
func normalizeBlockYAML(kind BlockKind, text string, indent int) (string, error) {
	trimmed := strings.TrimRight(strings.ReplaceAll(text, "\r\n", "\n"), " \t\n")
	if strings.TrimSpace(trimmed) == "" {
		return "", errors.New("块内容不能为空")
	}

	if err := validateBlock(kind, trimmed); err != nil {
		return "", err
	}

	lines := strings.Split(trimmed, "\n")

	// 计算公共基础缩进（忽略空行）
	base := -1
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		n := len(ln) - len(strings.TrimLeft(ln, " "))
		if base == -1 || n < base {
			base = n
		}
	}
	if base < 0 {
		base = 0
	}

	pad := strings.Repeat(" ", indent)
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			out = append(out, "")
			continue
		}
		out = append(out, pad+ln[base:])
	}

	first := -1
	for i, ln := range out {
		if strings.TrimSpace(ln) != "" {
			first = i
			break
		}
	}
	if first < 0 {
		return "", errors.New("块内容不能为空")
	}

	// 保证首行是序列元素（"- "），若客户端给的是裸映射则自行补上
	if !strings.HasPrefix(strings.TrimLeft(out[first], " "), "-") {
		out[first] = pad + "- " + strings.TrimLeft(out[first], " ")
		for i := first + 1; i < len(out); i++ {
			if strings.TrimSpace(out[i]) != "" {
				out[i] = "  " + out[i]
			}
		}
	}

	return strings.Join(out, "\n"), nil
}

// validateBlock 校验块内容是否是该 kind 下的单个合法元素
func validateBlock(kind BlockKind, text string) error {
	switch kind {
	case BlockItem:
		var items []Item
		if err := yaml.Unmarshal([]byte(text), &items); err == nil {
			if len(items) != 1 {
				return fmt.Errorf("该块只能包含一个条目，实际 %d 个", len(items))
			}
			return nil
		}
		var one Item
		if err := yaml.Unmarshal([]byte(text), &one); err != nil {
			return fmt.Errorf("YAML 格式错误: %s", yaml.FormatError(err, false, true))
		}
		return nil
	case BlockService:
		var svcs []Service
		if err := yaml.Unmarshal([]byte(text), &svcs); err == nil {
			if len(svcs) != 1 {
				return fmt.Errorf("该块只能包含一个分组，实际 %d 个", len(svcs))
			}
			return nil
		}
		var one Service
		if err := yaml.Unmarshal([]byte(text), &one); err != nil {
			return fmt.Errorf("YAML 格式错误: %s", yaml.FormatError(err, false, true))
		}
		return nil
	default:
		return fmt.Errorf("unknown block kind %q", kind)
	}
}

// findBlock 按 kind + 下标定位块
func findBlock(blocks []BlockInfo, kind BlockKind, serviceIndex, index int) (BlockInfo, bool) {
	for _, b := range blocks {
		if b.Kind != kind || b.Index != index {
			continue
		}
		if kind == BlockItem && b.ServiceIndex != serviceIndex {
			continue
		}
		return b, true
	}
	return BlockInfo{}, false
}

// lastBlockOf 找到目标序列中最后一个块，用于 insert 的锚点
func lastBlockOf(blocks []BlockInfo, kind BlockKind, serviceIndex int) (BlockInfo, bool) {
	var last BlockInfo
	found := false
	for _, b := range blocks {
		if b.Kind != kind {
			continue
		}
		if kind == BlockItem && b.ServiceIndex != serviceIndex {
			continue
		}
		if !found || b.Index > last.Index {
			last = b
			found = true
		}
	}
	return last, found
}

// applyPageBlock 对页面源码做文本级块替换 / 插入 / 删除，其余部分原样保留。
func applyPageBlock(content []byte, req BlockRequest) ([]byte, error) {
	blocks, err := listPageBlocks(content)
	if err != nil {
		// 定位失败时提示改用整文件编辑
		return nil, fmt.Errorf("%w；该文件无法按块定位，请使用「编辑原始 YAML」", err)
	}

	lines := splitLines(content)

	switch req.Action {
	case BlockUpdate:
		return applyBlockUpdate(lines, blocks, req)
	case BlockInsert:
		return applyBlockInsert(lines, blocks, req)
	case BlockDelete:
		return applyBlockDelete(lines, blocks, req)
	default:
		return nil, fmt.Errorf("unknown block action %q", req.Action)
	}
}

func applyBlockUpdate(lines []string, blocks []BlockInfo, req BlockRequest) ([]byte, error) {
	target, ok := findBlock(blocks, req.Kind, req.ServiceIndex, req.Index)
	if !ok {
		return nil, fmt.Errorf("未找到要修改的块（kind=%s, service_index=%d, index=%d）", req.Kind, req.ServiceIndex, req.Index)
	}

	text, err := normalizeBlockYAML(req.Kind, req.YAML, target.indent)
	if err != nil {
		return nil, err
	}

	start, end := target.LineRange[0], target.LineRange[1]
	out := make([]string, 0, len(lines))
	out = append(out, lines[:start-1]...)
	out = append(out, strings.Split(text, "\n")...)
	out = append(out, lines[end:]...)

	return []byte(strings.Join(out, "\n")), nil
}

func applyBlockInsert(lines []string, blocks []BlockInfo, req BlockRequest) ([]byte, error) {
	anchor, ok := lastBlockOf(blocks, req.Kind, req.ServiceIndex)
	if !ok {
		// 该分组还没有任何条目（items 为空）：需要单独定位插入点
		if req.Kind != BlockItem {
			return nil, errors.New("目标序列中没有可参照的块，请使用「编辑原始 YAML」")
		}
		return insertIntoEmptyItems(lines, req)
	}

	text, err := normalizeBlockYAML(req.Kind, req.YAML, anchor.indent)
	if err != nil {
		return nil, err
	}

	// 插到锚点块的最后一行之后
	at := anchor.LineRange[1]
	out := make([]string, 0, len(lines)+4)
	out = append(out, lines[:at]...)
	out = append(out, strings.Split(text, "\n")...)
	out = append(out, lines[at:]...)

	return []byte(strings.Join(out, "\n")), nil
}

// insertIntoEmptyItems 处理「分组存在但没有条目」的插入。
//
// items 可能写成 `items: []`（flow）或 `items:` 后无内容（block），
// 前者必须先替换掉那一行，否则会产出非法 YAML。
func insertIntoEmptyItems(lines []string, req BlockRequest) ([]byte, error) {
	content := []byte(strings.Join(lines, "\n"))

	body, err := parsePageAST(content)
	if err != nil {
		return nil, err
	}
	services := servicesSeq(body)
	if services == nil {
		return nil, errors.New("page yaml has no services list")
	}
	if req.ServiceIndex < 0 || req.ServiceIndex >= len(services.Values) {
		return nil, fmt.Errorf("分组下标 %d 超出范围", req.ServiceIndex)
	}

	svcNode, ok := services.Values[req.ServiceIndex].(*ast.MappingNode)
	if !ok {
		return nil, errors.New("分组节点不是映射")
	}

	field := mappingField(svcNode, "items")
	if field == nil {
		return nil, fmt.Errorf("分组 %d 没有 items 字段，请使用「编辑原始 YAML」", req.ServiceIndex)
	}

	keyTok := field.Key.GetToken()
	if keyTok == nil || keyTok.Position == nil {
		return nil, errors.New("无法定位 items 字段")
	}

	itemsLine := keyTok.Position.Line
	keyIndent := keyTok.Position.Column - 1
	itemIndent := keyIndent + 2

	text, err := normalizeBlockYAML(req.Kind, req.YAML, itemIndent)
	if err != nil {
		return nil, err
	}
	newLines := strings.Split(text, "\n")

	seq, _ := field.Value.(*ast.SequenceNode)
	isFlowEmpty := seq != nil && seq.IsFlowStyle

	out := make([]string, 0, len(lines)+len(newLines)+1)
	if isFlowEmpty {
		// 用块序列替换 `items: []`
		out = append(out, lines[:itemsLine-1]...)
		out = append(out, strings.Repeat(" ", keyIndent)+"items:")
		out = append(out, newLines...)
		out = append(out, lines[itemsLine:]...)
	} else {
		out = append(out, lines[:itemsLine]...)
		out = append(out, newLines...)
		out = append(out, lines[itemsLine:]...)
	}

	return []byte(strings.Join(out, "\n")), nil
}

func applyBlockDelete(lines []string, blocks []BlockInfo, req BlockRequest) ([]byte, error) {
	target, ok := findBlock(blocks, req.Kind, req.ServiceIndex, req.Index)
	if !ok {
		return nil, fmt.Errorf("未找到要删除的块（kind=%s, service_index=%d, index=%d）", req.Kind, req.ServiceIndex, req.Index)
	}

	start, end := target.LineRange[0], target.LineRange[1]

	// 顺带吃掉紧随其后的空行，避免留下连续空行
	if end < len(lines) && strings.TrimSpace(lines[end]) == "" {
		end++
	}

	out := make([]string, 0, len(lines))
	out = append(out, lines[:start-1]...)
	out = append(out, lines[end:]...)

	return []byte(strings.Join(out, "\n")), nil
}
