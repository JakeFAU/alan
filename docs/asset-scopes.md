# Binding retained documents to adapter capabilities

A host can provision one corpus for several agents. Each adapter request receives the documents its capability is bound to. A capability with no binding receives an empty selection. For example, a weather request can run alongside a document search in the same store.

The application author chooses a named capability in each `author.Effect`. The host supplies the documents and bindings before calling `runtime.Create`. Given a provision `p` and two `effects.Asset` values created with `effects.NewAsset`, the following grants search access to both documents and summary access to the guide.

```go
p.Assets = []effects.Asset{guide, handbook}
p.AssetScopes = map[string][]string{
    "search":  {guide.ID, handbook.ID},
    "summary": {guide.ID},
}
store, err := runtime.Create(ctx, path, p, runtime.DefaultLimits())
```

All effects that name the same capability share that binding. Give separate capabilities to effects that need different views. Creation rejects missing and duplicate IDs within a binding. The whole provision remains subject to the asset count and corpus byte limits.

The store retains each binding by ID and content-bound version. When preparing a request, the runtime checks the authored declaration and input schema. Before releasing selected bytes to an adapter, it checks source authority, the capability grant or delegation, and applicable composition limits. The adapter receives independent copies in asset ID order. A later revocation can prevent dispatch. A copy already released to a trusted, pure `Prepare` callback remains with that callback. Application-supplied fields in the ordinary effect payload travel under their own declared schema.

`runtime.InspectExecution` exposes the host bindings in `AssetScopes` and the retained corpus through `Asset(id)`. Each adapter attempt's `Prepared.Request.Assets` records the precise selection, including source, content and version. Reopening and replaying reconstruct that selection from the store without source files or adapter registration.

Fresh stores with documents or an explicit scope map use storage formats 17–30. Their underlying execution protocols correspond to formats 3–16, respectively. Earlier readers reject the scoped formats. Existing stores in formats 3–16 keep their historical whole-corpus preparation and replay semantics. To adopt capability-specific disclosure for new work, create a fresh store with explicit bindings. The quickstart binds its document to the `corpus` capability. [Quickstart source](../examples/quickstart/main.go)
