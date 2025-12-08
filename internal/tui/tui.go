package tui

import (
	"math"
	"os"
	"time"

	"navitui/internal/mpv"
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
			Margin(1)

	searchStyle = lipgloss.NewStyle().
			MarginBottom(1).
			BorderStyle(lipgloss.NormalBorder()).
			PaddingLeft(1).PaddingRight(1)

	tableStyle = lipgloss.NewStyle().
			MarginBottom(1).
			PaddingLeft(1).PaddingRight(1).
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
	"/_/ |_/\\__,_/ |___/_/ /_/  \\____/___/ \n"

type (
	onPlayFunc        func(id string) error
	getPlayerStatus   func() (*mpv.Status, error)
	tickNowPlayingMsg struct{}
)

type model struct {
	focus  focus
	search textinput.Model
	table  table.Model

	allSongs      []navidrome.Song
	filteredSongs []navidrome.Song

	playerStatus mpv.Status

	termHeight int
	termWidth  int

	onPlay        onPlayFunc
	getNowPlaying getPlayerStatus
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tickNowPlayingCmd(), textinput.Blink)
}

func tickNowPlayingCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return tickNowPlayingMsg{}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tickNowPlayingMsg:
		var cmds []tea.Cmd
		m.setPlayerStatus()

		cmds = append(cmds, tickNowPlayingCmd())

		return m, tea.Batch(cmds...)

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
			switch m.focus {
			case focusTable:
				idx := m.table.Cursor()
				song := m.filteredSongs[idx]
				if err := m.setNowPlaying(song); err != nil {
					return m, nil
				}
			case focusSearch:
				song := m.filteredSongs[0]
				if err := m.setNowPlaying(song); err != nil {
					return m, nil
				}
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
	header := headerStyle.
		Align(lipgloss.Center).
		Width(m.termWidth).
		Render(header)

	searchView := searchStyle.
		BorderForeground(lipgloss.Color("240")).
		Width(m.termWidth).
		Render(m.search.View())
	if m.focus == focusSearch {
		searchView = searchStyle.
			BorderForeground(lipgloss.Color("40")).
			Width(m.termWidth).
			Bold(true).
			Render(m.search.View())
	}

	tableView := tableStyle.
		BorderForeground(lipgloss.Color("240")).
		Width(m.termWidth).
		Render(m.table.View())

	if m.focus == focusTable {
		tableView = tableStyle.
			BorderForeground(lipgloss.Color("40")).
			Width(m.termWidth).
			Render(m.table.View())
	}

	nowPlayingView := lipgloss.JoinVertical(
		lipgloss.Center,
		lipgloss.NewStyle().Width(m.termWidth).Render("Now Playing: "),
		lipgloss.NewStyle().Width(m.termWidth).Render("Title: ", m.playerStatus.Metadata.Title),
		lipgloss.NewStyle().Render(m.playerStatus.Position.Round(time.Second).String()+" / "+m.playerStatus.Duration.Round(time.Second).String()),
		lipgloss.NewStyle().Width(m.termWidth).Render("Artist: ", m.playerStatus.Metadata.Artist),
		lipgloss.NewStyle().Width(m.termWidth).Render("Album: ", m.playerStatus.Metadata.Album),
	)

	views := []string{header, searchView, tableView, nowPlayingView}

	return lipgloss.JoinVertical(
		lipgloss.Center,
		views...,
	) + "\n"
}

func newModel(songs []navidrome.Song, op onPlayFunc, np getPlayerStatus) model {
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
		{Title: "Title", Width: getSize(width, .4)},
		{Title: "Artist", Width: getSize(width, .2)},
		{Title: "Album", Width: getSize(width, .15)},
		{Title: "Genre", Width: getSize(width, .15)},
	}

	m := model{
		focus:         focusSearch,
		search:        ti,
		allSongs:      songs,
		termHeight:    height - 2,
		termWidth:     width - 2,
		onPlay:        op,
		getNowPlaying: np,
	}
	m.buildTable(columns)

	return m
}

// use lithammer/fuzzy to find songs based on any attribute of the track. case-insensitive.
func matchTrack(t navidrome.Song, q string) bool {
	return fuzzy.MatchFold(q, t.Title) || fuzzy.MatchFold(q, t.Album) || fuzzy.MatchFold(q, t.Artist) || fuzzy.MatchFold(q, t.Genre)
}

// build table based on query
func (m *model) buildTable(columns []table.Column) {
	query := m.search.Value()

	m.filteredSongs = m.filteredSongs[:0]
	for _, t := range m.allSongs {
		if query == "" || matchTrack(t, query) {
			m.filteredSongs = append(m.filteredSongs, t)
		}
	}

	rows := make([]table.Row, len(m.filteredSongs))
	for i, t := range m.filteredSongs {
		rows[i] = table.Row{t.Title, t.Artist, t.Album, t.Genre}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(getSize(m.termHeight, .5)),
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

	m.table = t
}

func Run(songs []navidrome.Song, op onPlayFunc, np getPlayerStatus) error {
	m := newModel(songs, op, np)
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

// sets now playing to t
func (m *model) setNowPlaying(t navidrome.Song) error {
	if err := m.onPlay(t.ID); err != nil {
		return err
	}

	return nil
}

func (m *model) setPlayerStatus() error {
	status, err := m.getNowPlaying()
	if err != nil {
		return err
	}

	m.playerStatus = *status
	return nil
}

// helper function to get size based on % of s. Used for m.termHeight or m.termWidth
func getSize(s int, n float64) int {
	return int(math.Floor(float64(s) * n))
}
