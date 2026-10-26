package author_test

import (
	"github.com/JakeFAU/alan/author"
	"github.com/JakeFAU/alan/schema"
	"strings"
	"testing"
)

func TestApplicationValidatesWiringAndInputs(t *testing.T) {
	compile := func(d author.Definition) author.Artifact {
		t.Helper()
		a, e := author.Compile(d)
		if e != nil {
			t.Fatal(e)
		}
		return a
	}
	sender := compile(author.Definition{Inputs: textRecord("question"), Ports: map[string]author.Port{"worker": {Peer: "worker", Obligation: "q", Send: textRecord("q")}}, Steps: []author.Step{author.Send("worker", map[string]author.Expr{"q": author.Var("question")}), author.Success()}})
	worker := compile(author.Definition{Ports: map[string]author.Port{"caller": {Peer: "caller", Obligation: "q", Receive: textRecord("q")}}, Steps: []author.Step{author.Receive("caller", map[string]string{"q": "question"}), author.Success()}})
	app := author.Application{"caller": {Artifact: sender, Inputs: map[string]schema.Value{"question": schema.Text("why")}}, "worker": {Artifact: worker}}
	p, err := app.Provision()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Subjects) != 2 || len(p.Permissions) != 1 {
		t.Fatalf("unexpected provision %+v", p)
	}
	app["caller"] = author.Instance{Artifact: sender, Inputs: map[string]schema.Value{"question": schema.Int(1)}}
	if _, err = app.Provision(); err == nil {
		t.Fatal("wrong input accepted")
	}
	app["caller"] = author.Instance{Artifact: sender, Inputs: map[string]schema.Value{"question": schema.Text("why")}}
	delete(app, "worker")
	if _, err = app.Provision(); err == nil {
		t.Fatal("missing peer accepted")
	}
	app["worker"] = author.Instance{Artifact: compile(author.Definition{Ports: map[string]author.Port{"caller": {Peer: "caller", Obligation: "q", Receive: schema.Record{"q": {Kind: "int"}}}}, Steps: []author.Step{author.Receive("caller", map[string]string{"q": "n"}), author.Success()}})}
	if _, err = app.Provision(); err == nil {
		t.Fatal("incompatible wiring accepted")
	}
}

func TestApplicationRejectsInvalidUTF8Identity(t *testing.T) {
	a, err := author.Compile(author.Definition{Steps: []author.Step{author.Success()}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = (author.Application{string([]byte{0xfe}): {Artifact: a}, string([]byte{0xff}): {Artifact: a}}).Provision()
	if err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("want UTF-8 subject error, got %v", err)
	}
}
