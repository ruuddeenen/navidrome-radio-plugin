package template

import "testing"

func getter(m map[string]string) Getter {
	return func(field string) string { return m[field] }
}

func TestRenderDefaultTemplate(t *testing.T) {
	g := getter(map[string]string{"countrycode": "NL", "tags": "pop, rock", "name": "Free Radio"})
	got := Render("[{countrycode?:OTHER}] [{tags}] {name}", g)
	want := "[NL] [pop, rock] Free Radio"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRenderCountryFallbackAndEmptyTags(t *testing.T) {
	g := getter(map[string]string{"countrycode": "", "tags": "", "name": "X"})
	got := Render("[{countrycode?:OTHER}] [{tags}] {name}", g)
	if got != "[OTHER] X" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderEmptyTemplateFallsBackToName(t *testing.T) {
	g := getter(map[string]string{"name": "Only"})
	if got := Render("", g); got != "Only" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	g := getter(map[string]string{"a": "x", "b": "", "name": "N"})
	if got := Render("  {a}   {b}  {name} ", g); got != "x N" {
		t.Fatalf("got %q", got)
	}
}

func TestDefaultSubstitution(t *testing.T) {
	g := getter(map[string]string{"country": "Nederland", "name": "N"})
	if got := Render("{country?:Onbekend}", g); got != "Nederland" {
		t.Fatalf("got %q", got)
	}
	g = getter(map[string]string{"country": "", "name": "N"})
	if got := Render("{country?:Onbekend}", g); got != "Onbekend" {
		t.Fatalf("got %q", got)
	}
}
