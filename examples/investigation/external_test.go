package investigation_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestStandaloneModuleCompilesPublicComposition(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	module := "module external.example/investigation\n\ngo 1.27.3\n\nrequire github.com/JakeFAU/alan v0.0.0\nreplace github.com/JakeFAU/alan => " + root + "\n"
	source := `package main
import (
 "github.com/JakeFAU/alan/author"
 "github.com/JakeFAU/alan/examples/investigation"
)
func main() {
 for _, nested := range []bool{false,true} {
  c,err:=investigation.New("child","client","job",investigation.Budgets{MaxRevisions:1})
  if nested {c,err=investigation.Report("report","client","job","child",investigation.Budgets{MaxRevisions:1})}
  if err!=nil {panic(err)}
  bindings:=map[string]string{};for field:=range investigation.ResultSchema(){bindings[field]=field}
  a,err:=author.Compile(author.Definition{Name:"public client",Outputs:investigation.ResultSchema(),Ports:map[string]author.Port{"child":c.Port},Steps:[]author.Step{
   author.Send("child",map[string]author.Expr{"question":author.String("What is supported?")}),author.Receive("child",bindings),author.Success(),
  }})
  if err!=nil {panic(err)}
  c.Instances["client"]=author.Instance{Artifact:a}
  if _,err=c.Instances.Provision();err!=nil {panic(err)}
 }
}
`
	for name, content := range map[string]string{"go.mod": module, "main.go": source} {
		if err = os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	command := exec.CommandContext(t.Context(), "go", "build", "-mod=mod", "-o", filepath.Join(dir, "consumer"), ".")
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("external module build: %v\n%s", err, output)
	}
}
