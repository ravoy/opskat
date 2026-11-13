package desktop

import "testing"

// 菜单文案表是 mac 顶部菜单栏的展示契约：zh 前缀（含 SetLanguage 归一后的
// "zh-cn"）给中文，其余（含空默认）给英文。
func TestAppMenuLabelsFor(t *testing.T) {
	zh := AppMenuLabelsFor("zh-cn")
	if zh.Settings != "设置…" || zh.EditMenu != "编辑" || zh.Quit != "退出 OpsKat" || zh.CloseWindow != "关闭窗口" {
		t.Fatalf("zh-cn labels not Chinese: %+v", zh)
	}
	// i18next 原始写法也必须命中（调用方传的是 sys.Lang()，已小写；这里守住上层
	// 直接传原始码的容错）。
	if AppMenuLabelsFor("zh-CN").WindowMenu != "窗口" {
		t.Fatal("zh-CN (mixed case) should pick Chinese")
	}
	en := AppMenuLabelsFor("en")
	if en.Settings != "Settings…" || en.EditMenu != "Edit" || en.BringToFront != "Bring All to Front" {
		t.Fatalf("en labels not English: %+v", en)
	}
	if AppMenuLabelsFor("").Undo != "Undo" {
		t.Fatal("unset language must default to English")
	}
	// 两套文案都不允许空串：任何一项空都会让菜单栏出现空标签。
	for name, v := range map[string]string{
		"zh.settings": zh.Settings, "zh.pastePlain": zh.PastePlain,
		"en.selectAll": en.SelectAll, "en.minimize": en.Minimize,
	} {
		if v == "" {
			t.Fatalf("%s label is empty", name)
		}
	}
}
