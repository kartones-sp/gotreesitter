package gotreesitter

import "testing"

func TestSymbolByNameReturnsFirstDuplicate(t *testing.T) {
	lang := &Language{
		TokenCount:  5,
		SymbolNames: []string{"end", "identifier", "identifier", "stmt", "identifier"},
	}

	sym, ok := lang.SymbolByName("identifier")
	if !ok {
		t.Fatal("expected identifier symbol")
	}
	if sym != 1 {
		t.Fatalf("expected first identifier symbol 1, got %d", sym)
	}
}

func TestCanonicalSymbolMapsDuplicatesToFirst(t *testing.T) {
	lang := &Language{
		TokenCount:  5,
		SymbolNames: []string{"end", "identifier", "identifier", "stmt", "identifier"},
	}

	// All "identifier" symbols (1, 2, 4) should canonicalize to 1 (first occurrence).
	for _, sym := range []Symbol{1, 2, 4} {
		canonical := lang.CanonicalSymbol(sym)
		if canonical != 1 {
			t.Errorf("CanonicalSymbol(%d) = %d, want 1", sym, canonical)
		}
	}

	// Non-duplicate symbols map to themselves.
	if c := lang.CanonicalSymbol(0); c != 0 {
		t.Errorf("CanonicalSymbol(0) = %d, want 0", c)
	}
	if c := lang.CanonicalSymbol(3); c != 3 {
		t.Errorf("CanonicalSymbol(3) = %d, want 3", c)
	}

	// Out-of-range symbol returns itself.
	if c := lang.CanonicalSymbol(99); c != 99 {
		t.Errorf("CanonicalSymbol(99) = %d, want 99", c)
	}
}

func TestTokenSymbolsByNameFiltersTerminals(t *testing.T) {
	lang := &Language{
		TokenCount:  3,
		SymbolNames: []string{"end", "identifier", "identifier", "identifier", "stmt"},
	}

	syms := lang.TokenSymbolsByName("identifier")
	if len(syms) != 2 {
		t.Fatalf("expected 2 token symbols, got %d", len(syms))
	}
	if syms[0] != 1 || syms[1] != 2 {
		t.Fatalf("unexpected token symbols: %v", syms)
	}
}
