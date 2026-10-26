package lookup_test

import (
	"context"
	"github.com/JakeFAU/alan/adapters/lookup"
	"github.com/JakeFAU/alan/effects"
	"github.com/JakeFAU/alan/schema"
	"testing"
)

func TestRetainedExactByteSpan(t *testing.T) {
	a, e := effects.NewAsset("doc", "selected.txt", []byte("α first line\nSQLite retains durable records.\nlast"))
	if e != nil {
		t.Fatal(e)
	}
	r := effects.Request{ID: "i", Key: "k", Assets: []effects.Asset{a}, Payload: map[string]schema.Value{"query": schema.Text("durable records")}, Output: mustRetain(t, lookup.OutputSchema())}
	adapter := lookup.New()
	p, e := adapter.Prepare(r)
	if e != nil {
		t.Fatal(e)
	}
	a.Content = "changed"
	c := adapter.Execute(context.Background(), p)
	if c.Disposition != "validated" {
		t.Fatalf("%+v", c)
	}
	start, end := c.Payload["start"].Number, c.Payload["end"].Number
	if c.Payload["snippet"].Text != r.Assets[0].Content[start:end] || c.Payload["document"].Text != "doc" || c.Payload["version"].Text != r.Assets[0].Version {
		t.Fatal("inexact span")
	}
}
func mustRetain(t *testing.T, r schema.Record) schema.Retained {
	t.Helper()
	v, e := r.Retain()
	if e != nil {
		t.Fatal(e)
	}
	return v
}
