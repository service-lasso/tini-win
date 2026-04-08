package runner

import "testing"

func TestSplitCommandLine_PreservesQuotedPathsAndArgs(t *testing.T) {
	args, err := splitCommandLine(`"C:\Program Files\Tool\tool.exe" --flag "C:\Path With Spaces\file.txt"`)
	if err != nil {
		t.Fatalf("splitCommandLine error: %v", err)
	}
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %#v", len(args), args)
	}
	if args[0] != `C:\Program Files\Tool\tool.exe` {
		t.Fatalf("unexpected exe arg: %q", args[0])
	}
	if args[2] != `C:\Path With Spaces\file.txt` {
		t.Fatalf("unexpected quoted arg: %q", args[2])
	}
}

func TestSplitCommandLine_MultipleQuotedArgs(t *testing.T) {
	args, err := splitCommandLine(`"C:\Program Files\Tool\tool.exe" "alpha beta" "gamma delta"`)
	if err != nil {
		t.Fatalf("splitCommandLine error: %v", err)
	}
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %#v", len(args), args)
	}
	if args[1] != "alpha beta" || args[2] != "gamma delta" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestSplitCommandLine_UnterminatedQuote(t *testing.T) {
	_, err := splitCommandLine(`"C:\Program Files\Tool\tool.exe`)
	if err == nil {
		t.Fatalf("expected unterminated quote error")
	}
}
