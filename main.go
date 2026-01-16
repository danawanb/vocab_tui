package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
)

var (
	primaryColor   = lipgloss.Color("#FF6B9D")
	secondaryColor = lipgloss.Color("#C792EA")
	accentColor    = lipgloss.Color("#82AAFF")
	mutedColor     = lipgloss.Color("#697098")
	bgColor        = lipgloss.Color("#1E1E2E")

	titleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			MarginBottom(1)

	focusedStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	blurredStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	cursorStyle = focusedStyle

	menuItemStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color("#E0E0E0"))

	selectedMenuStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Foreground(primaryColor).
				Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			MarginTop(2)

	inputLabelStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true).
			MarginRight(1)

	buttonStyle = lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 3).
			Bold(true).
			MarginTop(1)

	buttonBlurredStyle = lipgloss.NewStyle().
				Background(mutedColor).
				Foreground(lipgloss.Color("#FFFFFF")).
				Padding(0, 3).
				MarginTop(1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E3A1")).
			Bold(true)
	docStyle = lipgloss.NewStyle().Margin(1, 2)

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()
)

type ScreenType int

const (
	MENU         ScreenType = 69
	ADD_VOCAB    ScreenType = 0
	TRAIN        ScreenType = 1
	REVIEW       ScreenType = 2
	VOCAB_LIST   ScreenType = 3
	VOCAB_DETAIL ScreenType = 4
)

type model struct {
	choices    []string
	cursor     int
	focusIndex int
	selected   map[int]struct{}
	screen     ScreenType
	textInputs []textinput.Model
	spinner    spinner.Model
	isLoading  bool
	//train
	trainVocabList []VocabLearn
	trainInputs    []textinput.Model
	trainStatus    int //0 mulai 1 ok 2 err dll
	trainTables    table.Model
	trainViewPort  viewport.Model
	trainVReady    bool
	db             *sqlx.DB
	//list
	listVocab      list.Model
	detailViewPort viewport.Model
	detailVReady   bool
	selectedVocab  VocabularyRes
}

type item struct {
	id, word, meaning string
}

func (i item) Title() string       { return i.word }
func (i item) Description() string { return i.meaning }
func (i item) FilterValue() string { return i.word }

func main() {
	cwd, _ := os.Getwd()
	log.Println("CWD:", cwd)
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)
	db, err := ConnectDB()
	if err != nil {
		log.Panic(err.Error())
	}

	res, err := SelectRandomVocab(db)

	if err != nil {
		log.Panic(err.Error())
	}
	items := []list.Item{}

	allVocab, err := AllVocabulary(db)
	if err != nil {
		log.Panic(err.Error())
	}

	log.Printf("loaded %d vocabulary items", len(allVocab))
	for _, voc := range allVocab {
		singleList := item{id: strconv.Itoa(voc.Id), word: voc.Word, meaning: voc.CoreMeaning}
		items = append(items, singleList)
	}

	log.Printf("created %d list items", len(items))

	m := model{
		choices:        []string{"Add Vocabulary", "Train", "Review", "Vocab List"},
		selected:       make(map[int]struct{}),
		screen:         MENU,
		textInputs:     make([]textinput.Model, 2),
		spinner:        s,
		isLoading:      false,
		trainInputs:    make([]textinput.Model, 5),
		trainVocabList: res,
		trainStatus:    0,
		trainTables:    table.New(),
		trainVReady:    false,
		trainViewPort:  viewport.New(80, 20),
		db:             db,
		listVocab:      list.New(items, list.NewDefaultDelegate(), 80, 25),
		detailViewPort: viewport.New(80, 20),
		detailVReady:   false,
	}
	m.listVocab.Title = "Vocabulary List"
	m.listVocab.SetShowStatusBar(true)
	m.listVocab.SetFilteringEnabled(true)

	var t textinput.Model

	for i := range m.textInputs {
		t = textinput.New()
		t.CharLimit = 255
		t.Width = 50

		switch i {
		case 0:
			t.SetValue("")
			t.Placeholder = "Vocabulary topics"
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.SetValue("")
			t.Placeholder = "Number of vocabulary"
		}

		m.textInputs[i] = t
	}

	for i := range m.trainInputs {
		t = textinput.New()
		t.CharLimit = 255
		t.Width = 50

		t.SetValue("")
		m.trainInputs[i] = t
	}

	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.screen {
	case MENU:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.choices)-1 {
					m.cursor++
				}
			case "enter", " ":
				m.selected[m.cursor] = struct{}{}
				switch m.cursor {
				case int(ADD_VOCAB):
					m.screen = ADD_VOCAB
					m.focusIndex = 0
					m.textInputs[0].Focus()
				case int(TRAIN):
					m.screen = TRAIN
				case int(REVIEW):
					m.screen = REVIEW
				case int(VOCAB_LIST):
					m.screen = VOCAB_LIST
				}
			}
		}

	case ADD_VOCAB:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			s := msg.String()
			switch s {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.screen = MENU
				m.focusIndex = 0
				m.cursor = 0
				for i := range m.textInputs {
					m.textInputs[i].SetValue("")
					m.textInputs[i].Blur()
				}
				return m, nil

			case "tab", "down":
				m.focusIndex++
				if m.focusIndex > len(m.textInputs) {
					m.focusIndex = 0
				}
				return m, m.updateFocus(ADD_VOCAB)

			case "shift+tab", "up":
				m.focusIndex--
				if m.focusIndex < 0 {
					m.focusIndex = len(m.textInputs)
				}
				return m, m.updateFocus(ADD_VOCAB)

			case "enter":
				if m.focusIndex == len(m.textInputs) {
					return m, m.submitForm(ADD_VOCAB)
				}
				if m.focusIndex == len(m.textInputs)-1 {
					return m, m.submitForm(ADD_VOCAB)
				}
				m.focusIndex++
				return m, m.updateFocus(ADD_VOCAB)
			}

		case spinner.TickMsg:
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd

		case doneMsg:
			m.isLoading = false
			m.focusIndex = 0
			m.screen = MENU
			for i := range m.textInputs {
				m.textInputs[i].SetValue("")
				m.textInputs[i].Blur()
			}
			// Reload vocabulary list after adding new vocabulary
			m.reloadVocabList()
			return m, nil
		}

		if !m.isLoading {
			cmd = m.updateInputs(msg, ADD_VOCAB)
		}
		return m, cmd

	case TRAIN:
		msgx := msg
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if m.trainStatus == 2 {
				switch msg.String() {
				case "ctrl+c", "q":
					return m, tea.Quit
				case "esc":
					m.screen = MENU
					m.trainStatus = 0
					for i := range m.trainInputs {
						m.trainInputs[i].SetValue("")
						m.trainInputs[i].Blur()
					}
					return m, nil
				case "enter":
					var curContent strings.Builder
					curContent.WriteString(m.trainTables.Rows()[m.trainTables.Cursor()][0])
					curContent.WriteString("\n\n")
					curContent.WriteString(m.trainTables.Rows()[m.trainTables.Cursor()][3])
					m.trainStatus = 3
					m.trainViewPort.SetContent(curContent.String())
					return m, nil

				case "up", "down", "k", "j":
					var cmd tea.Cmd
					//nav
					m.trainTables, cmd = m.trainTables.Update(msg)
					return m, cmd
				}
			} else {
				switch msg.String() {
				case "ctrl+c", "q":
					return m, tea.Quit
				case "tab", "down":
					m.focusIndex++
					if m.focusIndex > len(m.trainInputs) {
						m.focusIndex = 0
					}
					return m, m.updateFocus(TRAIN)

				case "shift+tab", "up":
					m.focusIndex--
					if m.focusIndex < 0 {
						m.focusIndex = len(m.trainInputs)
					}
					return m, m.updateFocus(TRAIN)
				case "enter":
					if m.focusIndex == len(m.trainInputs) {
						return m, m.submitForm(TRAIN)
					}
					m.focusIndex++
					if m.focusIndex == len(m.trainInputs) {
						m.focusIndex = 0
					}
					return m, m.updateFocus(TRAIN)

				case "esc":
					m.screen = MENU
					m.trainStatus = 0
					for i := range m.trainInputs {
						m.trainInputs[i].SetValue("")
						m.trainInputs[i].Blur()
					}

				}
			}
		case spinner.TickMsg:
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		case tableResultMsg:
			m.isLoading = false
			m.focusIndex = 0
			m.trainStatus = 2
			m.trainTables = msg.table
			m.trainVocabList = msg.vocabList

			for i := range m.trainInputs {
				m.trainInputs[i].SetValue("")
				m.trainInputs[i].Blur()
			}
			return m, nil

		case tea.WindowSizeMsg:
			if m.trainStatus == 3 {
				headerHeight := lipgloss.Height(m.headerView())
				footerHeight := lipgloss.Height(m.footerView())
				verticalMarginHeight := headerHeight + footerHeight

				if !m.trainVReady {
					m.trainViewPort = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
					m.trainViewPort.YPosition = headerHeight
					m.trainVReady = true
				} else {
					m.trainViewPort.Width = msg.Width
					m.trainViewPort.Height = msg.Height - verticalMarginHeight
				}
				switch msg := msgx.(type) {
				case tea.KeyMsg:
					switch msg.String() {
					case "ctrl+c", "q":
						return m, tea.Quit
					case "esc":
						m.trainStatus = 2
					}
				}

			}

		case doneMsg:
			m.isLoading = false
			m.focusIndex = 0
			m.screen = TRAIN
			m.trainStatus = 2
			for i := range m.trainInputs {
				m.trainInputs[i].SetValue("")
				m.trainInputs[i].Blur()
			}
			return m, nil

		case errMsg:
			m.isLoading = false
			m.focusIndex = 0
			m.screen = MENU

			for i := range m.trainInputs {
				m.trainInputs[i].SetValue("")
				m.trainInputs[i].Blur()
			}

			return m, nil

		}

		if !m.isLoading {
			cmd = m.updateInputs(msg, TRAIN)
		}
		return m, cmd
	case REVIEW:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "esc":
				m.screen = MENU
			}
		}

	case VOCAB_LIST:
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			h, v := docStyle.GetFrameSize()
			m.listVocab.SetSize(msg.Width-h, msg.Height-v)
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "esc":
				m.screen = MENU
				return m, nil
			case "enter":
				if it, ok := m.listVocab.SelectedItem().(item); ok {
					id, err := strconv.Atoi(it.id)
					if err == nil {
						detail, derr := GetVocabularyByID(m.db, id)
						if derr == nil {
							var b strings.Builder
							term := detail.Word
							if detail.POS != "" {
								term = term + " (" + detail.POS + ")"
							}
							b.WriteString(titleStyle.Render(term))
							b.WriteString("\n\n")
							b.WriteString(detail.CoreMeaning)

							if detail.CommonCollocations != "" {
								b.WriteString("\n\nCommon collocations:\n")
								b.WriteString(detail.CommonCollocations)
							}
							if detail.ExampleSentence != "" {
								b.WriteString("\n\nExample:\n")
								b.WriteString(detail.ExampleSentence)
							}
							if detail.Register != "" {
								b.WriteString("\n\nRegister: ")
								b.WriteString(detail.Register)
							}
							if detail.Notes != "" {
								b.WriteString("\n\nNotes:\n")
								b.WriteString(detail.Notes)
							}
							m.selectedVocab = detail
							m.detailViewPort.SetContent(b.String())
							m.screen = VOCAB_DETAIL
						}
					}
				}
				return m, nil
			}
		}

		var cmd tea.Cmd
		m.listVocab, cmd = m.listVocab.Update(msg)
		return m, cmd
	case VOCAB_DETAIL:
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			headerHeight := lipgloss.Height(m.headerViewDetail())
			footerHeight := lipgloss.Height(m.footerViewDetail())
			verticalMarginHeight := headerHeight + footerHeight

			if !m.detailVReady {
				m.detailViewPort = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
				m.detailViewPort.YPosition = headerHeight
				m.detailVReady = true
			} else {
				m.detailViewPort.Width = msg.Width
				m.detailViewPort.Height = msg.Height - verticalMarginHeight
			}
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "esc":
				m.screen = VOCAB_LIST
				return m, nil
			}
		}
		var vcmd tea.Cmd
		m.detailViewPort, vcmd = m.detailViewPort.Update(msg)
		return m, vcmd
	default:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m *model) updateFocus(menu ScreenType) tea.Cmd {
	switch menu {
	case ADD_VOCAB:
		cmds := make([]tea.Cmd, len(m.textInputs))
		for i := 0; i < len(m.textInputs); i++ {
			if i == m.focusIndex {
				cmds[i] = m.textInputs[i].Focus()
				m.textInputs[i].PromptStyle = focusedStyle
				m.textInputs[i].TextStyle = focusedStyle
			} else {
				m.textInputs[i].Blur()
				m.textInputs[i].PromptStyle = blurredStyle
				m.textInputs[i].TextStyle = lipgloss.NewStyle()
			}
		}
		return tea.Batch(cmds...)
	case TRAIN:
		cmds := make([]tea.Cmd, len(m.trainInputs))
		for i := 0; i < len(m.trainInputs); i++ {
			if i == m.focusIndex {
				cmds[i] = m.trainInputs[i].Focus()
				m.trainInputs[i].PromptStyle = focusedStyle
				m.trainInputs[i].TextStyle = focusedStyle
			} else {
				m.trainInputs[i].Blur()
				m.trainInputs[i].PromptStyle = blurredStyle
				m.trainInputs[i].TextStyle = lipgloss.NewStyle()
			}
		}
		return tea.Batch(cmds...)
	default:
		return tea.Batch()
	}
}

func (m *model) submitForm(menu ScreenType) tea.Cmd {
	m.isLoading = true

	var insert []Answer
	for i := range m.trainVocabList {
		ans := ""
		if i < len(m.trainInputs) {
			ans = m.trainInputs[i].Value()
		}
		singleInsert := Answer{
			Id:     m.trainVocabList[i].Id,
			Word:   m.trainVocabList[i].Word,
			Answer: ans,
		}
		insert = append(insert, singleInsert)
	}
	switch menu {
	case ADD_VOCAB:
		num, err := strconv.Atoi(m.textInputs[1].Value())
		if err != nil {
			return func() tea.Msg {
				return errMsg{}
			}
		}
		return tea.Batch(
			m.spinner.Tick,
			m.InsertThemeVocabulary(m.db, m.textInputs[0].Value(), num),
		)

	case TRAIN:
		return tea.Batch(
			m.spinner.Tick,
			m.InsertRandomVocab(m.db, insert),
		)

	default:
		return tea.Batch(
			m.spinner.Tick,
			fakeInsertCmd(),
		)

	}

}

func (m *model) updateInputs(msg tea.Msg, menu ScreenType) tea.Cmd {
	switch menu {
	case ADD_VOCAB:
		cmds := make([]tea.Cmd, len(m.textInputs))
		for i := range m.textInputs {
			m.textInputs[i], cmds[i] = m.textInputs[i].Update(msg)
		}
		return tea.Batch(cmds...)
	case TRAIN:
		cmds := make([]tea.Cmd, len(m.trainInputs))
		for i := range m.trainInputs {
			m.trainInputs[i], cmds[i] = m.trainInputs[i].Update(msg)
		}
		return tea.Batch(cmds...)
	default:
		return tea.Batch()
	}
}

func (m model) View() string {
	var s strings.Builder

	switch m.screen {
	case MENU:
		s.WriteString(titleStyle.Render("📚 Danawanb Vocabulary Learning App"))
		s.WriteString("\n\n")
		s.WriteString("What would you like to do?\n\n")

		for i, choice := range m.choices {
			cursor := "  "
			style := menuItemStyle
			if m.cursor == i {
				cursor = "▸ "
				style = selectedMenuStyle
			}
			s.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(choice)))
		}

		s.WriteString("\n")
		s.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • q: quit"))

	case ADD_VOCAB:
		s.WriteString(titleStyle.Render("➕ Add New Vocabulary"))
		s.WriteString("\n\n")

		if !m.isLoading {
			s.WriteString(inputLabelStyle.Render("Vocabulary:"))
			s.WriteString("\n")
			s.WriteString(m.textInputs[0].View())
			s.WriteString("\n\n")

			s.WriteString(inputLabelStyle.Render("Number:"))
			s.WriteString("\n")
			s.WriteString(m.textInputs[1].View())
			s.WriteString("\n\n")

			buttonText := "Submit"
			if m.focusIndex == len(m.textInputs) {
				s.WriteString(buttonStyle.Render(buttonText))
			} else {
				s.WriteString(buttonBlurredStyle.Render(buttonText))
			}

			s.WriteString("\n\n")
			s.WriteString(helpStyle.Render("enter: submit • tab/↑/↓: navigate • esc: back"))
		} else {
			s.WriteString("\n")
			s.WriteString(m.spinner.View())
			s.WriteString(" Saving vocabulary...")
		}

	case TRAIN:

		s.WriteString(titleStyle.Render("Training Mode"))
		switch m.trainStatus {
		case 0:
			s.WriteString("\n")
			if !m.isLoading {
				limit := len(m.trainVocabList)
				if len(m.trainInputs) < limit {
					limit = len(m.trainInputs)
				}
				for i := 0; i < limit; i++ {
					val := m.trainVocabList[i]
					s.WriteString(inputLabelStyle.Render(val.Word))
					s.WriteString(m.trainInputs[i].View())
					s.WriteString("\n")
				}

				s.WriteString("\n")
				buttonText := "Submit"
				if m.focusIndex == len(m.trainInputs) {
					s.WriteString(buttonStyle.Render(buttonText))
				} else {
					s.WriteString(buttonBlurredStyle.Render(buttonText))
				}

				s.WriteString("\n")
				s.WriteString(helpStyle.Render("enter: submit • tab/↑/↓: navigate • esc: back"))

			} else {
				s.WriteString(m.spinner.View())
				s.WriteString(" Checking answers...")

			}
		case 2:
			s.WriteString("\n")
			s.WriteString(m.trainTables.View())
			s.WriteString(helpStyle.Render("↑/↓: navigate table • esc: back to menu"))

		case 3:
			s.WriteString("\n")
			return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.trainViewPort.View(), m.footerView())
		}

	case REVIEW:
		s.WriteString(titleStyle.Render("Review Mode"))
		s.WriteString("\n\n")
		s.WriteString("Review feature coming soon!\n\n")
		s.WriteString(helpStyle.Render("esc: back to menu"))

	case VOCAB_LIST:
		s.WriteString(titleStyle.Render("All Vocabulary"))
		s.WriteString("\n")
		s.WriteString(m.listVocab.View())
		s.WriteString("\n")
		s.WriteString(helpStyle.Render("enter: open detail • esc: back to menu"))
	case VOCAB_DETAIL:
		return fmt.Sprintf("%s\n%s\n%s", m.headerViewDetail(), m.detailViewPort.View(), m.footerViewDetail())

	}

	return s.String()
}

type doneMsg struct{}
type errMsg struct{}

func fakeInsertCmd() tea.Cmd {
	return func() tea.Msg {

		time.Sleep(2 * time.Second)
		return doneMsg{}
	}
}

func (m *model) InsertThemeVocabulary(db *sqlx.DB, theme string, num int) tea.Cmd {
	return func() tea.Msg {
		err := InsertVocabulary(db, theme, num)
		if err != nil {
			return func() tea.Msg {
				return errMsg{}
			}
		}
		return doneMsg{}
	}
}

func (m *model) InsertRandomVocab(db *sqlx.DB, val []Answer) tea.Cmd {
	return func() tea.Msg {
		res, err := JawabRandomVocab(val)
		if err != nil {
			return func() tea.Msg {
				return errMsg{}
			}
		}

		nextVocab, err := SelectRandomVocab(db)
		if err != nil {
			return func() tea.Msg {
				return errMsg{}
			}
		}

		return tableResultMsg{table: newAnswerTable(res), vocabList: nextVocab}
	}
}

type tableResultMsg struct {
	table     table.Model
	vocabList []VocabLearn
}

func newAnswerTable(values []Answer) table.Model {
	columns := []table.Column{
		{Title: "Words", Width: 25},
		{Title: "Answer", Width: 25},
		{Title: "IsCorrect", Width: 10},
		{Title: "Meaning", Width: 85},
	}

	rows := []table.Row{}

	for _, val := range values {
		isCorrectStr := "false"
		if val.IsCorrect {
			isCorrectStr = "true"
		}
		singleRow := table.Row{val.Word, val.Answer, isCorrectStr, val.RealAnswer}
		rows = append(rows, singleRow)
	}

	tt := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(8),
	)

	st := table.DefaultStyles()
	st.Header = st.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	st.Selected = st.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	tt.SetStyles(st)

	return tt
}

func (m model) headerView() string {
	title := titleStyle.Render("Details")
	line := strings.Repeat("─", max(0, m.trainViewPort.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m model) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.trainViewPort.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.trainViewPort.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (m model) headerViewDetail() string {
	title := titleStyle.Render("Vocabulary Detail")
	line := strings.Repeat("─", max(0, m.detailViewPort.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m model) footerViewDetail() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.detailViewPort.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.detailViewPort.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (m *model) reloadVocabList() {
	allVocab, err := AllVocabulary(m.db)
	if err != nil {
		log.Printf("Error reloading vocab list: %v", err)
		return
	}

	items := []list.Item{}
	for _, voc := range allVocab {
		singleList := item{id: strconv.Itoa(voc.Id), word: voc.Word, meaning: voc.CoreMeaning}
		items = append(items, singleList)
	}

	m.listVocab.SetItems(items)
	log.Printf("Reloaded %d vocabulary items", len(items))
}
