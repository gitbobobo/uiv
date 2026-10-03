package store

import (
	"context"
	"errors"
	"os"
	"testing"
)

func commit(t *testing.T, s *Store, name string) File {
	t.Helper()
	tmp, err := s.TempFile()
	if err != nil {
		t.Fatal(err)
	}
	tmp.WriteString("data")
	tmp.Close()
	f, err := s.Commit(context.Background(), tmp.Name(), File{Ext: "png", Name: name, Size: 4, Type: "image/png"})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestCommitGetDelete(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()

	f := commit(t, s, "a.png")
	if len(f.ID) != 10 {
		t.Fatalf("id %q should be 10 chars", f.ID)
	}
	got, err := s.Get(ctx, f.ID)
	if err != nil || got != f {
		t.Fatalf("Get = %+v, %v; want %+v", got, err, f)
	}
	if _, err := os.Stat(s.Path(f)); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, f.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.Path(f)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("file should be gone, stat err = %v", err)
	}
	if _, err := s.Get(ctx, f.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after delete = %v, want ErrNotFound", err)
	}
	if err := s.Delete(ctx, f.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second Delete = %v, want ErrNotFound", err)
	}
}

func TestListPaginates(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()

	for range 5 {
		commit(t, s, "x.png")
	}

	var seen []File
	cursor := ""
	for {
		page, next, err := s.List(ctx, cursor, 2)
		if err != nil {
			t.Fatal(err)
		}
		seen = append(seen, page...)
		if next == "" {
			break
		}
		cursor = next
	}
	if len(seen) != 5 {
		t.Fatalf("saw %d files, want 5", len(seen))
	}
	unique := map[string]bool{}
	for i, f := range seen {
		unique[f.ID] = true
		if i > 0 {
			prev := seen[i-1]
			if f.CreatedAt.After(prev.CreatedAt) || (f.CreatedAt.Equal(prev.CreatedAt) && f.ID > prev.ID) {
				t.Fatalf("not newest first at %d: %+v after %+v", i, f, prev)
			}
		}
	}
	if len(unique) != 5 {
		t.Fatalf("duplicate ids across pages: %+v", seen)
	}
}

func TestListRejectsBadCursor(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, _, err := s.List(context.Background(), "garbage", 10); err == nil {
		t.Fatal("expected error")
	}
}
