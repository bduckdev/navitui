package tui

import (
	"math"
	"os"

	"navitui/internal/navidrome"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("57")).
			MarginBottom(1)

	searchStyle = lipgloss.NewStyle().
			MarginBottom(1).
			BorderStyle(lipgloss.NormalBorder()).
			PaddingLeft(1).PaddingRight(1)

	tableStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder())
)

type focus int

const (
	focusSearch focus = iota
	focusTable
)

var header = "" +
	"    _   __            _ ________  ______\n" +
	"   / | / /___ __   __(_)_  __/ / / /  _/\n" +
	"  /  |/ / __ `/ | / / / / / / / / // /  \n" +
	" / /|  / /_/ /| |/ / / / / / /_/ // /   \n" +
	"/_/ |_/\\__,_/ |___/_/ /_/  \\____/___/\n"

type model struct {
	focus  focus
	search textinput.Model
	table  table.Model

	allTracks []navidrome.Track
	filtered  []navidrome.Track

	nowPlaying navidrome.Track
	isPlaying  bool

	termHeight int
	termWidth  int
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "/":
			m.focus = focusSearch
			m.search.Focus()
			m.table.Blur()
			return m, nil

		case "tab", "ctrl+w":
			if m.focus == focusSearch {
				m.focus = focusTable
				m.table.Focus()
				m.search.Blur()
				return m, nil
			} else {
				m.focus = focusSearch
				m.search.Focus()
				m.table.Blur()
				return m, nil
			}

		case "esc":
			m.focus = focusTable
			m.table.Focus()
			m.search.Blur()
			return m, nil

		case "enter":
			if m.focus == focusTable && len(m.filtered) > 0 {
				idx := m.table.Cursor()
				track := m.filtered[idx]
				m.setNowPlaying(track)
				m.isPlaying = true

				return m, nil
			}
		}
	}

	switch m.focus {
	case focusSearch:
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		cmds = append(cmds, cmd)

		m.buildTable(m.table.Columns())
	case focusTable:
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)

	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	header := headerStyle.Width(m.termWidth).Render(header)

	searchView := searchStyle.BorderForeground(lipgloss.Color("240")).Width(m.termWidth).Render(m.search.View())
	if m.focus == focusSearch {
		searchView = searchStyle.BorderForeground(lipgloss.Color("40")).Width(m.termWidth).Bold(true).Render(m.search.View())
	}

	tableView := tableStyle.BorderForeground(lipgloss.Color("240")).Width(m.termWidth).Render(m.table.View())
	if m.focus == focusTable {
		tableView = tableStyle.BorderForeground(lipgloss.Color("40")).Width(m.termWidth).Render(m.table.View())
	}

	nowPlayingView := lipgloss.JoinVertical(
		lipgloss.Center,
		lipgloss.NewStyle().Width(m.termWidth).Render("Now Playing: "),
		lipgloss.NewStyle().Width(m.termWidth).Render("Title: ", m.nowPlaying.Title),
		lipgloss.NewStyle().Width(m.termWidth).Render("Artist: ", m.nowPlaying.Artist),
		lipgloss.NewStyle().Width(m.termWidth).Render("Album: ", m.nowPlaying.Album),
	)

	views := []string{header, searchView, tableView}

	if m.isPlaying {
		views = append(views, nowPlayingView)
	}

	return lipgloss.JoinVertical(
		lipgloss.Center,
		views...,
	) + "\n"
}

func newModel(tracks []navidrome.Track) model {
	width, height, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		width, height = 500, 500
	}

	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Focus()
	ti.Prompt = "> "
	ti.CharLimit = 128
	ti.Width = 60

	columns := []table.Column{
		{Title: "Title", Width: int(math.Floor(float64(width) * 0.50))},
		{Title: "Artist", Width: int(math.Floor(float64(width) * 0.20))},
		{Title: "Album", Width: int(math.Floor(float64(width) * 0.15))},
		{Title: "Genre", Width: int(math.Floor(float64(width) * 0.10))},
	}

	m := model{
		focus:      focusSearch,
		search:     ti,
		allTracks:  tracks,
		termHeight: height - 2,
		termWidth:  width - 2,
	}
	m.buildTable(columns)

	return m
}

func matchTrack(t navidrome.Track, q string) bool {
	return fuzzy.Match(q, t.Title) || fuzzy.Match(q, t.Album) || fuzzy.Match(q, t.Artist) || fuzzy.Match(q, t.Genre)
}

// build table based on query
func (m *model) buildTable(columns []table.Column) {
	query := m.search.Value()

	m.filtered = m.filtered[:0]
	for _, t := range m.allTracks {
		if query == "" || matchTrack(t, query) {
			m.filtered = append(m.filtered, t)
		}
	}

	rows := make([]table.Row, len(m.filtered))
	for i, t := range m.filtered {
		rows[i] = table.Row{t.Title, t.Artist, t.Album, t.Genre}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
		table.WithWidth(m.termWidth),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)
	t.SetStyles(s)

	m.table = t
}

func Run(tracks []navidrome.Track) error {
	m := newModel(tracks)
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

func (m *model) setNowPlaying(t navidrome.Track) {
	m.nowPlaying = t
}
