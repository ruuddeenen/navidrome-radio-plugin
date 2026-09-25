module github.com/ruuddeenen/navidrome-radio-sync-plugin

go 1.25

require github.com/navidrome/navidrome/plugins/pdk/go v0.0.0

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/extism/go-pdk v1.1.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// The Navidrome plugin PDK is not published as a versioned module. The Makefile
// clones Navidrome at the pinned version into .build/navidrome and points this
// replace at its PDK submodule.
replace github.com/navidrome/navidrome/plugins/pdk/go => ./.build/navidrome/plugins/pdk/go
