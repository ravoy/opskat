// Package desktop owns native desktop integration shared by domain binders.
package desktop

import "github.com/wailsapp/wails/v3/pkg/application"

func OpenFile(options application.OpenFileDialogOptions) (string, error) {
	options.CanChooseFiles = true
	options.Window = application.Get().Window.Current()
	return application.Get().Dialog.OpenFileWithOptions(&options).PromptForSingleSelection()
}

func OpenFiles(options application.OpenFileDialogOptions) ([]string, error) {
	options.CanChooseFiles = true
	options.Window = application.Get().Window.Current()
	return application.Get().Dialog.OpenFileWithOptions(&options).PromptForMultipleSelection()
}

func OpenDirectory(options application.OpenFileDialogOptions) (string, error) {
	options.CanChooseDirectories = true
	options.Window = application.Get().Window.Current()
	return application.Get().Dialog.OpenFileWithOptions(&options).PromptForSingleSelection()
}

func SaveFile(options application.SaveFileDialogOptions) (string, error) {
	options.CanCreateDirectories = true
	options.Window = application.Get().Window.Current()
	return application.Get().Dialog.SaveFileWithOptions(&options).PromptForSingleSelection()
}
