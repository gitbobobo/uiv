package config

import "testing"

func TestParseSize(t *testing.T) {
	cases := map[string]int64{
		"1024":  1024,
		"200MB": 200 << 20,
		"200mb": 200 << 20,
		"1 GB":  1 << 30,
		"512K":  512 << 10,
		"10B":   10,
	}
	for in, want := range cases {
		got, err := ParseSize(in)
		if err != nil || got != want {
			t.Errorf("ParseSize(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", "abc", "-1MB", "0", "1.5GB"} {
		if _, err := ParseSize(in); err == nil {
			t.Errorf("ParseSize(%q) should fail", in)
		}
	}
}

func TestFromEnvRequiresToken(t *testing.T) {
	t.Setenv("UIV_TOKEN", "")
	if _, err := FromEnv(); err == nil {
		t.Fatal("expected error without UIV_TOKEN")
	}
}

func TestFromEnvDefaults(t *testing.T) {
	t.Setenv("UIV_TOKEN", "secret")
	t.Setenv("UIV_PUBLIC_URL", "https://uiv.example.com/")
	c, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if c.PublicURL != "https://uiv.example.com" || c.DataDir != "/data" || c.Addr != ":8080" || c.MaxSize != 200<<20 {
		t.Fatalf("unexpected config: %+v", c)
	}
}

func TestFromEnvRejectsBadPublicURL(t *testing.T) {
	t.Setenv("UIV_TOKEN", "secret")
	t.Setenv("UIV_PUBLIC_URL", "uiv.example.com")
	if _, err := FromEnv(); err == nil {
		t.Fatal("expected error for URL without scheme")
	}
}
