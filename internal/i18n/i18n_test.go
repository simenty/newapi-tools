// NewAPI Tools - i18n framework tests
package i18n

import (
	"os"
	"testing"
)

func TestNewBundle(t *testing.T) {
	bundle, err := NewBundle("zh-CN")
	if err != nil {
		t.Fatalf("NewBundle(zh-CN) failed: %v", err)
	}
	if bundle.lang != "zh-CN" {
		t.Errorf("expected lang zh-CN, got %s", bundle.lang)
	}
	if len(bundle.messages) == 0 {
		t.Error("expected non-empty messages map")
	}
}

func TestNewBundleEn(t *testing.T) {
	bundle, err := NewBundle("en")
	if err != nil {
		t.Fatalf("NewBundle(en) failed: %v", err)
	}
	if bundle.lang != "en" {
		t.Errorf("expected lang en, got %s", bundle.lang)
	}
	if len(bundle.messages) == 0 {
		t.Error("expected non-empty messages map")
	}
}

func TestNewBundleFallback(t *testing.T) {
	// Unsupported language should fall back to zh-CN
	bundle, err := NewBundle("fr-FR")
	if err != nil {
		t.Fatalf("NewBundle(fr-FR) should fallback to zh-CN, got error: %v", err)
	}
	if bundle.lang != "fr-FR" {
		t.Errorf("expected lang fr-FR (preserved), got %s", bundle.lang)
	}
	// Should have zh-CN messages since that's the fallback
	if len(bundle.messages) == 0 {
		t.Error("expected non-empty messages from fallback")
	}
}

func TestNewBundleInvalidLang(t *testing.T) {
	// Requesting zh-CN explicitly and it doesn't exist would error,
	// but zh-CN always exists in our embedded FS, so test a truly missing case
	// by testing that the fallback mechanism works
	bundle, err := NewBundle("xx-XX")
	if err != nil {
		t.Fatalf("NewBundle should fallback for unknown language, got error: %v", err)
	}
	// Verify we got messages from the fallback
	if bundle.T("install.success") == "install.success" {
		t.Error("fallback should have loaded zh-CN messages")
	}
}

func TestT(t *testing.T) {
	bundle, err := NewBundle("zh-CN")
	if err != nil {
		t.Fatalf("NewBundle failed: %v", err)
	}

	got := bundle.T("install.success")
	if got == "install.success" {
		t.Error("expected translated string, got key back")
	}
}

func TestTMissing(t *testing.T) {
	bundle, err := NewBundle("zh-CN")
	if err != nil {
		t.Fatalf("NewBundle failed: %v", err)
	}

	got := bundle.T("nonexistent.key.that.does.not.exist")
	if got != "nonexistent.key.that.does.not.exist" {
		t.Errorf("expected key itself for missing translation, got %q", got)
	}
}

func TestTWithArgs(t *testing.T) {
	bundle, err := NewBundle("zh-CN")
	if err != nil {
		t.Fatalf("NewBundle failed: %v", err)
	}

	// Test with a key that has a placeholder
	got := bundle.T("install.pulling_image", "myimage:latest")
	if got == "install.pulling_image" {
		t.Error("expected translated string with args, got key back")
	}
}

func TestPackageTWithoutInit(t *testing.T) {
	// Save and restore defaultBundle
	saved := defaultBundle
	defaultBundle = nil
	defer func() { defaultBundle = saved }()

	// T() with nil defaultBundle should return the key itself
	got := T("some.key")
	if got != "some.key" {
		t.Errorf("expected key when defaultBundle is nil, got %q", got)
	}
}

func TestInit(t *testing.T) {
	// Save and restore defaultBundle
	saved := defaultBundle
	defer func() { defaultBundle = saved }()

	if err := Init("en"); err != nil {
		t.Fatalf("Init(en) failed: %v", err)
	}
	if defaultBundle == nil {
		t.Fatal("expected defaultBundle to be initialized")
	}
	if defaultBundle.lang != "en" {
		t.Errorf("expected lang en, got %s", defaultBundle.lang)
	}

	got := T("install.success")
	if got == "install.success" {
		t.Error("expected translated string, got key back")
	}
}

func TestInitEmptyLang(t *testing.T) {
	// Save and restore defaultBundle and env vars
	saved := defaultBundle
	defer func() { defaultBundle = saved }()

	// Clear relevant env vars to test default fallback
	os.Unsetenv("LC_ALL")
	os.Unsetenv("LANG")

	if err := Init(""); err != nil {
		t.Fatalf("Init('') failed: %v", err)
	}
	if defaultBundle == nil {
		t.Fatal("expected defaultBundle to be initialized")
	}
	// Should default to zh-CN when no env vars are set
	if defaultBundle.lang != "zh-CN" {
		t.Errorf("expected default lang zh-CN, got %s", defaultBundle.lang)
	}
}

func TestInitEnvLang(t *testing.T) {
	// Save and restore defaultBundle and env vars
	saved := defaultBundle
	defer func() { defaultBundle = saved }()

	savedLCALL := os.Getenv("LC_ALL")
	savedLANG := os.Getenv("LANG")
	defer func() {
		os.Setenv("LC_ALL", savedLCALL)
		os.Setenv("LANG", savedLANG)
	}()

	os.Unsetenv("LC_ALL")
	os.Setenv("LANG", "en_US.UTF-8")

	if err := Init(""); err != nil {
		t.Fatalf("Init('') with LANG=en_US.UTF-8 failed: %v", err)
	}
	if defaultBundle.lang != "en" {
		t.Errorf("expected lang en from LANG env, got %s", defaultBundle.lang)
	}
}

func TestFlattenYAML(t *testing.T) {
	raw := map[string]any{
		"install": map[string]any{
			"success": "安装成功！",
			"pulling": "正在拉取镜像...",
		},
		"status": map[string]any{
			"running": "运行中",
		},
	}
	result := make(map[string]string)
	flattenYAML(raw, "", result)

	if result["install.success"] != "安装成功！" {
		t.Errorf("expected '安装成功！', got %q", result["install.success"])
	}
	if result["install.pulling"] != "正在拉取镜像..." {
		t.Errorf("expected '正在拉取镜像...', got %q", result["install.pulling"])
	}
	if result["status.running"] != "运行中" {
		t.Errorf("expected '运行中', got %q", result["status.running"])
	}
}

func TestNormalizeLang(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"zh_CN.UTF-8", "zh-CN"},
		{"en_US.UTF-8", "en"},
		{"zh-CN", "zh-CN"},
		{"en", "en"},
		{"zh_TW.Big5", "zh-CN"},
		{"en_GB.UTF-8", "en"},
		{"ja_JP.UTF-8", "zh-CN"}, // unsupported defaults to zh-CN
	}
	for _, c := range cases {
		got := normalizeLang(c.input)
		if got != c.expected {
			t.Errorf("normalizeLang(%q) = %q, want %q", c.input, got, c.expected)
		}
	}
}

func TestBundleLangMethod(t *testing.T) {
	bundle, err := NewBundle("en")
	if err != nil {
		t.Fatalf("NewBundle failed: %v", err)
	}
	if bundle.Lang() != "en" {
		t.Errorf("Lang() = %q, want %q", bundle.Lang(), "en")
	}
}
