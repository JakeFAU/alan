package owningtree

import "github.com/JakeFAU/alan/author"

func flag(reg string, v bool) author.Step { return author.Set(reg, author.Boolean(v)) }
func decision(value string) author.Step   { return author.Set("decision", author.String(value)) }

// A guard may only turn tree_valid off. In particular, an invalid left
// selection cannot terminate the caller or bypass the original right Await.
func requireEqual(reg string, want author.Expr) author.Step {
	return author.IfEqual(author.Var(reg), want, nil, []author.Step{flag("tree_valid", false)})
}

func validate(c branch, paths []string) []author.Step {
	var present []author.Step
	for _, p := range paths {
		present = append(present, validateLeaf(c.prefix+p, c.name+"_answer")...)
	}
	envelope := func(status, reason string) []author.Step {
		return []author.Step{requireEqual(c.name+"_status", author.String(status)), requireEqual(c.name+"_reason", author.String(reason))}
	}
	diagnosis := append(envelope("failed", "member_failed"), flag("tree_answer", false))
	present = append(present, author.IfEqual(author.Var(c.name+"_answer"), author.Boolean(true), envelope("succeeded", "member_succeeded"), diagnosis))
	return []author.Step{flag(c.name+"_answer", true), author.IfEqual(author.Var(c.name+"_present"), author.Boolean(true), present, []author.Step{flag("tree_valid", false)})}
}

func validateLeaf(path, childAnswer string) []author.Step {
	tag := "report_" + path + "classification"
	diagnosis := []author.Step{flag(childAnswer, false), requireEqual("seen", author.Boolean(false)),
		author.IfEqual(author.Var(tag), author.String("candidate_failed"), nil, []author.Step{requireEqual(tag, author.String("candidate_missing_reply"))})}
	return []author.Step{
		requireEqual("report_"+path+"id", author.Var(path+"id")),
		author.IfEqual(author.Var(path+"value"), author.Integer(0), []author.Step{author.Set("expected", author.Integer(1))}, []author.Step{author.Set("expected", author.Integer(0))}),
		flag("seen", false),
		author.ForEach(author.Var("report_"+path+"answers"), map[string]string{"id": "row_id", "value": "row_value"},
			requireEqual("seen", author.Boolean(false)), flag("seen", true),
			requireEqual("row_id", author.Var(path+"id")), requireEqual("row_value", author.Var("expected"))),
		author.IfEqual(author.Var(tag), author.String("answer"), []author.Step{requireEqual("seen", author.Boolean(true))}, diagnosis),
	}
}
