package releases

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pidanou/helmtui/styles"
)

type selectedView int

const (
	releasesView selectedView = iota
	historyView
	notesView
	metadataView
	hooksView
	valuesView
	manifestView
)

type Model struct {
	selectedView selectedView
	keys         []keyMap
	help         help.Model
	releaseTable table.Model
	historyTable table.Model
	notesVP      viewport.Model
	metadataVP   viewport.Model
	hooksVP      viewport.Model
	valuesVP     viewport.Model
	manifestVP   viewport.Model
	textInput    textinput.Model
	width        int
	height       int
}

func InitModel() (tea.Model, tea.Cmd) {
	t, h := generateTables()
	k := generateKeys()
	ti := textinput.New()
	m := Model{releaseTable: t, historyTable: h, help: help.New(), keys: k, textInput: ti}

	m.releaseTable.Focus()
	return m, nil
}

func (m Model) Init() tea.Cmd {
	return m.list
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	if m.textInput.Focused() {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				cmd = m.upgrade(m.textInput.Value())
				m.textInput.SetValue("")
				m.textInput.Blur()
				return m, cmd
			case "esc":
				m.textInput.SetValue("")
				m.textInput.Blur()
			}
		}
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	}
	switch m.selectedView {
	case releasesView:
		m.releaseTable.Focus()
		m.releaseTable, cmd = m.releaseTable.Update(msg)
		cmds = append(cmds, cmd)
	case historyView:
		m.historyTable, cmd = m.historyTable.Update(msg)
		cmds = append(cmds, cmd)
	case notesView:
		m.notesVP, cmd = m.notesVP.Update(msg)
		cmds = append(cmds, cmd)
	case metadataView:
		m.metadataVP, cmd = m.metadataVP.Update(msg)
		cmds = append(cmds, cmd)
	case hooksView:
		m.hooksVP, cmd = m.hooksVP.Update(msg)
		cmds = append(cmds, cmd)
	case valuesView:
		m.valuesVP, cmd = m.valuesVP.Update(msg)
		cmds = append(cmds, cmd)
	case manifestView:
		m.manifestVP, cmd = m.manifestVP.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height // -6: remove the table padding and menu
		m.setTable(&m.releaseTable, releaseCols, m.width, m.height)
		m.setTable(&m.historyTable, historyCols, m.width, 7) // 7: 5 rows + 2 rows for the header
		m.notesVP = viewport.New(m.width-6, 0)
		m.metadataVP = viewport.New(m.width-6, 0)
		m.hooksVP = viewport.New(m.width-6, 0)
		m.valuesVP = viewport.New(m.width-6, 0)
		m.manifestVP = viewport.New(m.width-6, 0)
		m.help.Width = msg.Width
	case listMsg:
		m.releaseTable.SetRows(msg.content)
		m.releaseTable, cmd = m.releaseTable.Update(msg)
		cmds = append(cmds, cmd, m.history, m.getNotes, m.getMetadata, m.getHooks, m.getValues, m.getManifest)
	case historyMsg:
		m.historyTable.SetRows(msg.content)
		m.historyTable.SetCursor(0)
		m.historyTable, cmd = m.historyTable.Update(msg)
		cmds = append(cmds, cmd)
	case upgradeMsg:
		cmds = append(cmds, m.list)
		m.selectedView = releasesView
	case deleteMsg:
		cmds = append(cmds, m.list)
		m.releaseTable.SetCursor(0)
		m.selectedView = releasesView
	case rollbackMsg:
		cmds = append(cmds, m.history)
		m.historyTable.SetCursor(0)
	case notesMsg:
		m.notesVP.SetContent(msg.content)
		m.notesVP, cmd = m.notesVP.Update(msg)
		cmds = append(cmds, cmd)
	case metadataMsg:
		m.metadataVP.SetContent(msg.content)
		m.metadataVP, cmd = m.metadataVP.Update(msg)
		cmds = append(cmds, cmd)
	case hooksMsg:
		m.hooksVP.SetContent(msg.content)
		m.hooksVP, cmd = m.hooksVP.Update(msg)
		cmds = append(cmds, cmd)
	case valuesMsg:
		m.valuesVP.SetContent(msg.content)
		m.valuesVP, cmd = m.valuesVP.Update(msg)
		cmds = append(cmds, cmd)
	case templatesMsg:
		m.manifestVP.SetContent(msg.content)
		m.manifestVP, cmd = m.manifestVP.Update(msg)
		cmds = append(cmds, cmd)
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			switch m.selectedView {
			case releasesView:
				cmds = append(cmds, m.list)
			}
		case "R":
			switch m.selectedView {
			case historyView:
				return m, m.rollback
			}
		case "d":
			return m, m.delete
		case "u":
			m.textInput.Placeholder = "Enter a chart name or chart directory (absolute path)"
			cmd = m.textInput.Focus()
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
			// return m, m.upgrade
		case "esc", "backspace":
			switch m.selectedView {
			case releasesView:
			default:
				m.historyTable.SetCursor(0)
				m.selectedView = releasesView
				m.historyTable.Blur()
				m.releaseTable = releaseTableCache
			}
		case "enter", " ":
			switch m.selectedView {
			case releasesView:
				m.selectedView = historyView
				releaseTableCache = m.releaseTable
				m.releaseTable.SetHeight(3)
				m.releaseTable.SetRows([]table.Row{m.releaseTable.SelectedRow()})
				m.releaseTable.GotoTop()
				m.historyTable.Focus()
				cmds = append(cmds, m.history, m.getNotes, m.getMetadata, m.getHooks, m.getValues, m.getManifest)
			}
		case "tab":
			switch m.selectedView {
			case releasesView:
			case manifestView:
				m.selectedView = historyView
			default:
				m.selectedView++
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	header := m.renderReleasesTableView() + "\n" + m.menuView()
	remainingHeight := m.height - lipgloss.Height(header) + lipgloss.Height(m.menuView()) - 2 - 1 // releaseTable padding + helper
	if m.textInput.Focused() {
		remainingHeight = remainingHeight - lipgloss.Height(m.textInput.View())
	}
	view := ""
	switch m.selectedView {
	case releasesView:
		tHeight := m.height - 2 - 1 // releaseTable padding + helper
		if m.textInput.Focused() {
			tHeight--
		}
		m.releaseTable.SetHeight(tHeight) // -2: releaseTable padding, -1: helper
		view = m.renderReleasesTableView()
	case historyView:
		m.historyTable.SetHeight(remainingHeight - 2)
		view = header + "\n" + m.renderHistoryTableView()
	case notesView:
		m.notesVP.Height = remainingHeight - 4
		view = header + "\n" + m.renderNotesView()
	case metadataView:
		m.metadataVP.Height = remainingHeight - 4 // -4: 2*1 Padding + 2 borders
		view = header + "\n" + m.renderMetadataView()
	case hooksView:
		m.hooksVP.Height = remainingHeight - 4 // -4: 2*1 Padding + 2 borders
		view = header + "\n" + m.renderHooksView()
	case valuesView:
		m.valuesVP.Height = remainingHeight - 4 // -4: 2*1 Padding + 2 borders
		view = header + "\n" + m.renderValuesView()
	case manifestView:
		m.manifestVP.Height = remainingHeight - 4 // -4: 2*1 Padding + 2 borders
		view = header + "\n" + m.renderManifestView()
	}
	if m.textInput.Focused() {
		view += "\n" + m.textInput.View()
	}
	helpView := m.help.View(m.keys[m.selectedView])
	return view + "\n" + strings.Repeat(" ", m.width-lipgloss.Width(helpView)) + helpView
}

func (m Model) menuView() string {
	menuItem := []string{
		"History",
		"Notes",
		"Metadata",
		"Hooks",
		"Values",
		"Manifest",
	}
	doc := strings.Builder{}

	var renderedTabs []string

	for i, t := range menuItem {
		var style lipgloss.Style
		isFirst, isActive := i == 0, i == int(m.selectedView)-1
		if isActive {
			style = styles.ActiveTabStyle
		} else {
			style = styles.InactiveTabStyle
		}
		border, _, _, _, _ := style.GetBorder()
		if isFirst && isActive {
			border.BottomLeft = "│"
		} else if isFirst && !isActive {
			border.BottomLeft = "├"
		}
		style = style.Border(border)
		renderedTabs = append(renderedTabs, style.Render(t))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	doc.WriteString(row + strings.Repeat("─", m.width-lipgloss.Width(row)-1) + styles.Border.TopRight)
	return doc.String()
}

func (m Model) renderReleasesTableView() string {
	var releasesTopBorder string
	tableView := m.releaseTable.View()
	var baseStyle lipgloss.Style
	releasesTopBorder = styles.GenerateTopBorderWithTitle(" Releases ", m.releaseTable.Width(), styles.Border, styles.InactiveStyle)
	baseStyle = styles.InactiveStyle.Border(styles.Border, false, true, true)
	tableView = baseStyle.Render(tableView)
	return lipgloss.JoinVertical(lipgloss.Top, releasesTopBorder, tableView)
}

func (m Model) renderHistoryTableView() string {
	tableView := m.historyTable.View()
	var baseStyle lipgloss.Style
	baseStyle = styles.InactiveStyle.Border(styles.Border).UnsetBorderTop()
	tableView = baseStyle.Render(tableView)
	return tableView
}

func (m Model) renderNotesView() string {
	view := m.notesVP.View()
	var baseStyle lipgloss.Style
	baseStyle = styles.InactiveStyle.Padding(1, 2).Border(styles.Border, false, true, true)
	view = baseStyle.Render(view)
	return view
}

func (m Model) renderMetadataView() string {
	view := m.metadataVP.View()
	baseStyle := styles.InactiveStyle.Padding(1, 2).Border(styles.Border, false, true, true)
	view = baseStyle.Render(view)
	return view
}

func (m Model) renderHooksView() string {
	view := m.hooksVP.View()
	baseStyle := styles.InactiveStyle.Padding(1, 2).Border(styles.Border, false, true, true)
	view = baseStyle.Render(view)
	return view
}

func (m Model) renderValuesView() string {
	view := m.valuesVP.View()
	baseStyle := styles.InactiveStyle.Padding(1, 2).Border(styles.Border, false, true, true)
	view = baseStyle.Render(view)
	return view
}

func (m Model) renderManifestView() string {
	view := m.manifestVP.View()
	baseStyle := styles.InactiveStyle.Padding(1, 2).Border(styles.Border, false, true, true)
	view = baseStyle.Render(view)
	return view
}

func (m Model) SetHeight(height int) {
	m.height = height
}

func (m Model) SetWidth(width int) {
	m.width = width
}
