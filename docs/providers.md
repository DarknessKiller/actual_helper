# Custom Providers

Providers are statically compiled into the binary and registered at startup
(`cmd/app/main.go` factories map). They are always active; configuration is
tuning only (exclude/include keywords, category rules, account mappings) loaded
via `POST /config` — it never activates or deactivates a provider.

A reference implementation lives in `internal/providers/sample/`. It implements
the full provider contract (`providers.Provider`, `providers.ConfigurableProvider`
and `io.Closer`) against the same seam the real providers use, with deterministic
synthetic parsing (no network, no credentials, no real data). It is **not**
registered by default.

## Enabling the sample provider

Add the import and factory entry to `cmd/app/main.go`:

```go
import (
	// ...
	sampleprov "actual_helper/internal/providers/sample"
	// ...
)

func main() {
	registry, loader, env := bootstrap.Init(map[string]bootstrap.ProviderFactory{
		"tng":        tngprov.New,
		"ryt":        rytprov.New,
		"hsbccredit": hsbccreditprov.New,
		"hlb":        hlbprov.New,
		"gxbank":     gxbankprov.New,
		"uobcredit":  uobcreditprov.New,
		"sample":     sampleprov.New, // <-- add this line
	})
	// ...
}
```

Rebuild. The server now serves `POST /convert/sample` like any other provider.

## Configuring the sample provider

Load a config via `POST /config`. The merged (global + `providers.sample`)
rules are pushed to the sample provider through `ConfigurableProvider.Reload`
on the next request, exactly like the built-in providers:

```json
{
  "global": { "exclude_keywords": [], "include_keywords": [], "categories": [] },
  "providers": {
    "sample": {
      "exclude_keywords": ["noise"],
      "categories": [
        { "keyword": "coffee", "group": "Food & Dining", "category": "Cafe" }
      ]
    }
  }
}
```

To clear tuning, call `DELETE /config`; every `ConfigurableProvider` (sample
included) is reloaded with empty rules. To fetch the sample config file served
by `PROVIDER_CONFIG_PATH`, call `GET /config`.

## Writing your own provider

Mirror `internal/providers/sample/service.go`:

1. Implement `providers.Provider` (`Name`, `ParseCSV`, `ParsePDFText`,
   `ExtractionMethod`).
2. Implement `providers.ConfigurableProvider.Reload(...)` so `POST /config` /
   `DELETE /config` tuning takes effect with zero restart.
3. Optionally implement `io.Closer` if the provider owns resources to release.
4. Add a `New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider` factory matching `bootstrap.ProviderFactory`.
5. Register it in the `cmd/app/main.go` factories map.

Do not add HTTP concepts or file-format detection to the provider layer — the
service layer owns those.
