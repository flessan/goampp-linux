package uikit

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ProjectsPage is the Windows project-creation and project-list hierarchy.
// Platform adapters supply the form, list and the appropriate description of
// their creation workflow; the list must provide its own scrolling.
func ProjectsPage(creation, list fyne.CanvasObject, description string) fyne.CanvasObject {
	newProject := TitledCard("New project", description, nil, creation)
	projects := TitledCard("Your projects", "", nil, list)
	return container.NewBorder(container.NewVBox(newProject, Gap(1, 8)), nil, nil, nil, projects)
}

// EmptyProjects is the common empty-list treatment.
func EmptyProjects() fyne.CanvasObject {
	return container.NewCenter(container.NewVBox(
		container.NewCenter(widget.NewIcon(theme.FolderIcon())),
		container.NewCenter(MutedText("No projects yet — create one above.", 13)),
	))
}

// ProjectRow uses the Windows project identity and actions alignment.
func ProjectRow(title, detail, actions fyne.CanvasObject) fyne.CanvasObject {
	return container.NewBorder(nil, nil,
		container.NewHBox(Gap(4, 1), container.NewVBox(title, detail)), actions)
}

// StartupPage shares the Windows startup cards and their relative layout.
// The introduction and service grid come from the platform integration.
func StartupPage(platformTitle string, platform, services fyne.CanvasObject, serviceTitle, serviceDescription string) fyne.CanvasObject {
	platformCard := TitledCard(platformTitle, "", nil, platform)
	serviceCard := TitledCard(serviceTitle, serviceDescription, nil, services)
	return container.NewBorder(container.NewVBox(platformCard, Gap(1, 8)), nil, nil, nil, serviceCard)
}

// SettingsPage lays out the reference sections. An optional behavior section
// allows desktop-specific capabilities such as a Linux tray preference.
func SettingsPage(appearance, paths, tools, about, behavior fyne.CanvasObject) fyne.CanvasObject {
	sections := []fyne.CanvasObject{
		SectionLabel("APPEARANCE"), Surface(appearance, 2, 2, 0, 0),
		SectionLabel("PATHS"), Surface(paths, 2, 2, 0, 0),
		SectionLabel("TOOLS"), Surface(tools, 2, 2, 0, 0),
		SectionLabel("ABOUT"), Surface(about, 2, 2, 0, 0),
	}
	if behavior != nil {
		sections = append(sections, SectionLabel("BEHAVIOR"), Surface(behavior, 2, 2, 0, 0))
	}
	return container.NewVScroll(container.NewVBox(sections...))
}

// StartupServiceCard aligns a service logo with its platform-provided controls.
func StartupServiceCard(avatar, content, actions fyne.CanvasObject) fyne.CanvasObject {
	return Surface(container.NewBorder(nil, nil, avatar, actions, content), 10, 10, 12, 12)
}

// ProjectForm provides the reference field order. A domain control is only
// supplied when the platform can configure a domain during creation.
func ProjectForm(framework, name, domain fyne.CanvasObject) fyne.CanvasObject {
	fields := []fyne.CanvasObject{widget.NewLabel("Framework"), framework, widget.NewLabel("Name"), name}
	if domain != nil {
		fields = append(fields, widget.NewLabel("Domain"), domain)
	}
	return container.New(layout.NewFormLayout(), fields...)
}

// SettingsPathRow presents a path with the Windows label and folder action.
func SettingsPathRow(label, path string, open func()) fyne.CanvasObject {
	button := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), open)
	button.Importance = widget.LowImportance
	value := widget.NewLabel(path)
	value.Truncation = fyne.TextTruncateEllipsis
	value.Importance = widget.LowImportance
	return Pad(container.NewBorder(nil, nil, MinSize(widget.NewLabel(label), 120, 0), button, value), 0, 0, 10, 4)
}

// VirtualHostForm is the Windows field order and sizing. Domain and document
// root controls may include platform-specific selectors or browse buttons.
func VirtualHostForm(domain, documentRoot, port, server fyne.CanvasObject) fyne.CanvasObject {
	return container.New(layout.NewFormLayout(),
		widget.NewLabel("Domain"), domain,
		widget.NewLabel("Document root"), documentRoot,
		widget.NewLabel("Port"), container.NewBorder(nil, nil, MinSize(port, 90, 0), nil),
		widget.NewLabel("Server"), MinSize(server, 140, 0),
	)
}

// EmptyVirtualHosts is shown in the domain list when there are no entries.
func EmptyVirtualHosts() fyne.CanvasObject {
	return container.NewCenter(MutedText("No virtual hosts yet — add one on the right.", 13))
}

// VirtualHostEditor shares the reference card and form spacing; optional
// supplemental controls remain supplied by the platform integration.
func VirtualHostEditor(title fyne.CanvasObject, description string, form, actions, extra fyne.CanvasObject) fyne.CanvasObject {
	content := []fyne.CanvasObject{title, MutedText(description, 11.5), Gap(1, 4), form, Gap(1, 6), actions}
	if extra != nil {
		content = append(content, extra)
	}
	return Surface(container.NewVBox(content...), 10, 10, 12, 12)
}
