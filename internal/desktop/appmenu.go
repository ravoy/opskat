package desktop

import (
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// AppMenuLabels 是应用菜单的全部文案。标准角色（Undo/Cut/Quit…）在 wails 里是
// 硬编码英文，macOS 也不会代为本地化——要跟随应用内语言（zh-CN/en），只能整体
// 自建并逐项重标。集中成一张表，两种语言一眼对齐，漏译在编译期可见。
type AppMenuLabels struct {
	Settings     string
	About        string
	Hide         string
	HideOthers   string
	ShowAll      string
	Quit         string
	EditMenu     string
	Undo         string
	Redo         string
	Cut          string
	Copy         string
	Paste        string
	PastePlain   string
	Delete       string
	SelectAll    string
	WindowMenu   string
	Minimize     string
	Zoom         string
	BringToFront string
	CloseWindow  string
}

// AppMenuLabelsFor 按语言取全套菜单文案。lang 与 system binder 同源（i18next 写法
// "zh-CN"/"en"，SetLanguage 已统一小写），判断方式与 i18n.Pick 一致：zh 前缀中文。
func AppMenuLabelsFor(lang string) AppMenuLabels {
	zh := strings.HasPrefix(strings.ToLower(lang), "zh")
	pick := func(z, e string) string {
		if zh {
			return z
		}
		return e
	}
	return AppMenuLabels{
		Settings:     pick("设置…", "Settings…"),
		About:        pick("关于 OpsKat", "About OpsKat"),
		Hide:         pick("隐藏 OpsKat", "Hide OpsKat"),
		HideOthers:   pick("隐藏其它", "Hide Others"),
		ShowAll:      pick("显示全部", "Show All"),
		Quit:         pick("退出 OpsKat", "Quit OpsKat"),
		EditMenu:     pick("编辑", "Edit"),
		Undo:         pick("撤销", "Undo"),
		Redo:         pick("重做", "Redo"),
		Cut:          pick("剪切", "Cut"),
		Copy:         pick("拷贝", "Copy"),
		Paste:        pick("粘贴", "Paste"),
		PastePlain:   pick("粘贴为纯文本", "Paste and Match Style"),
		Delete:       pick("删除", "Delete"),
		SelectAll:    pick("全选", "Select All"),
		WindowMenu:   pick("窗口", "Window"),
		Minimize:     pick("最小化", "Minimize"),
		Zoom:         pick("缩放", "Zoom"),
		BringToFront: pick("全部置于前", "Bring All to Front"),
		CloseWindow:  pick("关闭窗口", "Close Window"),
	}
}

// relabel 给角色项换标签：角色保留原生行为与系统快捷键。wails 在非 darwin 平台
// 会给部分角色返回 nil（本文件只在 darwin 调用），nil 时保持 wails 自带英文标签。
func relabel(menu *application.Menu, role application.Role, label string) {
	if item := menu.FindByRole(role); item != nil {
		item.SetLabel(label)
	}
}

// InstallAppMenu 组装并安装原生应用菜单（macOS 屏幕顶部菜单栏；其它平台无操作：
// Windows/Linux 的菜单长在窗口标题栏里，本项目窗口自带 UI）。自定义项：设置（⌘，
// 打开设置页外观分区）、关于（设置页 about 分区）；其余为标准角色，标签按应用语言。
// 返回的 rebuild 供语言切换时整体重建（SetApplicationMenu 原地替换）；ok 为 false
// 表示本平台无菜单。
func InstallAppMenu(appName string, lang func() string, openSettings func(section string)) (rebuild func(), ok bool) {
	if runtime.GOOS != "darwin" {
		return func() {}, false
	}
	build := func() {
		L := AppMenuLabelsFor(lang())
		menu := application.NewMenu()

		appSub := menu.AddSubmenu(appName)
		settings := appSub.Add(L.Settings)
		settings.SetAccelerator("CmdOrCtrl+,")
		settings.OnClick(func(*application.Context) { openSettings("appearance") })
		about := appSub.Add(L.About)
		about.OnClick(func(*application.Context) { openSettings("about") })
		appSub.AddSeparator()
		appSub.AddRole(application.Hide)
		relabel(appSub, application.Hide, L.Hide)
		appSub.AddRole(application.HideOthers)
		relabel(appSub, application.HideOthers, L.HideOthers)
		appSub.AddRole(application.UnHide)
		relabel(appSub, application.UnHide, L.ShowAll)
		appSub.AddSeparator()
		appSub.AddRole(application.Quit)
		relabel(appSub, application.Quit, L.Quit)

		editSub := menu.AddSubmenu(L.EditMenu)
		editSub.AddRole(application.Undo)
		relabel(editSub, application.Undo, L.Undo)
		editSub.AddRole(application.Redo)
		relabel(editSub, application.Redo, L.Redo)
		editSub.AddSeparator()
		editSub.AddRole(application.Cut)
		relabel(editSub, application.Cut, L.Cut)
		editSub.AddRole(application.Copy)
		relabel(editSub, application.Copy, L.Copy)
		editSub.AddRole(application.Paste)
		relabel(editSub, application.Paste, L.Paste)
		editSub.AddRole(application.PasteAndMatchStyle)
		relabel(editSub, application.PasteAndMatchStyle, L.PastePlain)
		editSub.AddRole(application.Delete)
		relabel(editSub, application.Delete, L.Delete)
		editSub.AddRole(application.SelectAll)
		relabel(editSub, application.SelectAll, L.SelectAll)

		winSub := menu.AddSubmenu(L.WindowMenu)
		// 最小化角色的常量是 wails 的导出名（英式拼写），无法更名，标签在下方重标。
		winSub.AddRole(application.Minimise)              //nolint:misspell // wails 的常量名
		relabel(winSub, application.Minimise, L.Minimize) //nolint:misspell // wails 的常量名
		winSub.AddRole(application.Zoom)
		relabel(winSub, application.Zoom, L.Zoom)
		winSub.AddSeparator()
		winSub.AddRole(application.Front)
		relabel(winSub, application.Front, L.BringToFront)
		winSub.AddRole(application.CloseWindow)
		relabel(winSub, application.CloseWindow, L.CloseWindow)

		application.Get().Menu.SetApplicationMenu(menu)
	}
	build()
	return build, true
}
