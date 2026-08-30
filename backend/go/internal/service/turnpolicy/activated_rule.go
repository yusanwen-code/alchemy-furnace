package turnpolicy

// ActivatedPillRule 本轮激活的金丹规则(spec §6.5;每颗金丹 ≤2 心智模型/≤3 启发式/≤1 示例)
type ActivatedPillRule struct {
	PillID             string
	PillName           string
	MentalModels       []string // "名称：one_liner"
	DecisionHeuristics []string // "若 condition 则 action(例: case)"
	ExampleDialogues   []string // "问:user 答:assistant"
}
