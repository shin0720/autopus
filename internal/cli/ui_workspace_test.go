package cli

import "testing"

func TestResolveWorkspacePathForOS_WindowsDriveRoot(t *testing.T) {
	t.Parallel()

	got, err := resolveWorkspacePathForOS("windows", `C:\Users\SAMSUNG\autopus`, `/mnt/e`)
	if err != nil {
		t.Fatalf("resolveWorkspacePathForOS returned error: %v", err)
	}
	if got != `E:\` {
		t.Fatalf("expected E:\\, got %q", got)
	}
}

func TestResolveWorkspacePathForOS_WindowsAbsolute(t *testing.T) {
	t.Parallel()

	got, err := resolveWorkspacePathForOS("windows", `C:\Users\SAMSUNG\autopus`, `C:\Users\SAMSUNG\Desktop`)
	if err != nil {
		t.Fatalf("resolveWorkspacePathForOS returned error: %v", err)
	}
	if got != `C:\Users\SAMSUNG\Desktop` {
		t.Fatalf("expected desktop path, got %q", got)
	}
}

func TestResolveWorkspacePathForOS_WindowsRelative(t *testing.T) {
	t.Parallel()

	got, err := resolveWorkspacePathForOS("windows", `C:\Users\SAMSUNG\autopus`, `internal`)
	if err != nil {
		t.Fatalf("resolveWorkspacePathForOS returned error: %v", err)
	}
	if got != `C:\Users\SAMSUNG\autopus\internal` {
		t.Fatalf("expected relative workspace path, got %q", got)
	}
}

func TestFirstLine(t *testing.T) {
	t.Parallel()

	got := firstLine("\nwarning line\nsecond line\n")
	if got != "warning line" {
		t.Fatalf("expected first non-empty line, got %q", got)
	}
}
