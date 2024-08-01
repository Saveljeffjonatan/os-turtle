package bubbletea

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"

	"log"
	util "turtle/utils"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type AppState int

const (
	CheckingAuth AppState = iota
	DisplayTable
	TitleInput
	DescriptionInput
	TicketInput
	ReviewInput
	MergeRequestSummary AppState = iota
	ErrorState
)

type reviewInput struct {
	Label    string
	Selected bool
	Value    int
}

type MergeRequestCreation struct {
	TitleInput          textinput.Model
	DescriptionInput    textarea.Model
	TicketInput         textinput.Model
	ReviewInput         []reviewInput
	MergeRequestSummary int32
}

type MergeRequest struct {
	Title     string
	Author    string
	CreatedAt time.Time
}

type Model struct {
	mergeRequests []MergeRequest
	cursor        int
	state         AppState
	table         table.Model
	creation      MergeRequestCreation
	usr           util.Author
	err           error
}

type usr struct{ Author util.Author }

type errMsg struct{ error }

func (e errMsg) Error() string { return e.error.Error() }

func NewModel(mrs []MergeRequest) Model {
	model := Model{
		state:         DisplayTable,
		mergeRequests: mrs,
		cursor:        0,
	}

	model.initTable()
	model.initInputs()

	return model
}

func (m *Model) initTable() {
	columns := []table.Column{
		{Title: "Title", Width: 45},
		{Title: "Author", Width: 20},
		{Title: "Created", Width: 20},
	}

	t := table.New(table.WithColumns(columns))

	var rows []table.Row
	for _, mr := range m.mergeRequests {
		title := util.TruncateString(mr.Title, 45)
		createdAt := util.TimeSince(mr.CreatedAt)

		rows = append(rows, []string{title, mr.Author, createdAt})
	}

	t.SetRows(rows)
	t.SetHeight(5)

	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("23")).BorderBottom(true).Bold(false)
	s.Selected = s.Selected.Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(false)

	t.SetStyles(s)
	t.Focus()

	m.table = t
}

func (m *Model) initInputs() {
	titleInput := textinput.New()
	titleInput.Placeholder = "Enter title"
	titleInput.Focus()
	titleInput.Cursor.Blink = true
	titleInput.CharLimit = 256
	titleInput.Width = 256

	descriptionInput := textarea.New()
	descriptionInput.Placeholder = "Buddha, what makes us human? \n Selecting all images with traffic lights."
	descriptionInput.Focus()
	descriptionInput.CharLimit = 600
	descriptionInput.SetWidth(85)

	// Input your review users here
	// TODO: Make this dynamic
	// The value is your gitlab users ID
	reviewInput := []reviewInput{
		{Label: "Test user", Selected: false, Value: 1234567},
	}

	ticketInput := textinput.New()
	ticketInput.Placeholder = "Enter title"
	ticketInput.Focus()
	ticketInput.Cursor.Blink = true
	ticketInput.CharLimit = 4
	ticketInput.Width = 4

	m.creation = MergeRequestCreation{
		TitleInput:       titleInput,
		DescriptionInput: descriptionInput,
		TicketInput:      ticketInput,
		ReviewInput:      reviewInput,
	}
}

func (m Model) Init() tea.Cmd {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
		return tea.Quit
	}

	return checkServer
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

	case usr:
		m.usr = msg.Author
		m.state = DisplayTable
		return m, nil

	case errMsg:
		m.err = msg
		m.state = ErrorState
		return m, nil
	}

	switch m.state {
	case DisplayTable:
		return m.updateDisplayTableState(msg)

	case TitleInput, DescriptionInput, TicketInput, ReviewInput, MergeRequestSummary:
		return m.updateMergeInput(msg)
	}

	return m, cmd
}

func (m Model) updateDisplayTableState(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	sourceBranch := util.GetGitBranch()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+y":
			m.state = TitleInput
			m.creation.TitleInput.SetValue(sourceBranch)
			m.creation.TitleInput.Focus()
			return m, nil
		}

		m.table, cmd = m.table.Update(msg)
	}

	return m, cmd
}

func (m Model) updateMergeInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.state {
	case TitleInput:
		m.creation.TitleInput, cmd = m.creation.TitleInput.Update(msg)
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyCtrlY {
			m.state = DescriptionInput
			m.creation.DescriptionInput.Focus()
			return m, nil
		}

	case DescriptionInput:
		m.creation.DescriptionInput, cmd = m.creation.DescriptionInput.Update(msg)
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyCtrlY {
			m.state = TicketInput
			m.creation.DescriptionInput.Focus()
			return m, nil
		}

	case TicketInput:
		m.creation.TicketInput, cmd = m.creation.TicketInput.Update(msg)
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyCtrlY {
			m.state = ReviewInput
			m.cursor = 1
			return m, nil
		}

	case ReviewInput:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.cursor > 1 {
					m.cursor--
				}

			case "down", "j":
				if m.cursor < len(m.creation.ReviewInput)-1 {
					m.cursor++
				}

			case "enter":
				m.creation.ReviewInput[m.cursor].Selected = !m.creation.ReviewInput[m.cursor].Selected

			case "ctrl+y":
				m.state = MergeRequestSummary
			}
		}

	case MergeRequestSummary:
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyCtrlY {
			sourceBranch := util.GetGitBranch()
			title := m.creation.TitleInput.Value()
			description := m.creation.DescriptionInput.Value()
			ticket := m.creation.TicketInput.Value()

			// We use hapoPattern at my job so I created it with that syntax
			// You can change this to your own ticket pattern
			// There is a few places in this code that you will need to change
			hapoPattern := regexp.MustCompile(`hapo[\s\-0-9]+`)

			if !hapoPattern.MatchString(sourceBranch) {
				title = fmt.Sprintf("[Hapo-%s] - %s", ticket, title)
			}

			var reviewerIDs []int
			for _, review := range m.creation.ReviewInput {
				if review.Selected {
					reviewerIDs = append(reviewerIDs, review.Value)
				}
			}

			assigneeID := int32(m.usr.Id)

			projectPath := os.Getenv("GITLAB_PROJECT")

			apiPath := fmt.Sprintf("%s/merge_requests", projectPath)
			fmt.Println("apiPath: ", apiPath)
			err := util.CreateGitlabMergeRequest(apiPath, description, ticket, title, assigneeID, reviewerIDs)
			if err != nil {
				fmt.Println("Error creating merge request:", err)
				return m, nil
			}

			return m, tea.Quit
		}
	}

	return m, cmd
}

func (m Model) View() string {
	var view string

	switch m.state {
	case CheckingAuth:
		view = fmt.Sprintln("Checking Authentification: ")

	case DisplayTable:
		view = m.table.View()

	case TitleInput:
		view = "Enter title \n" + m.creation.TitleInput.View()

	case DescriptionInput:
		view = "Enter description \n" + m.creation.DescriptionInput.View()

	case TicketInput:
		view = "Enter [hapo-]:ID \n" + m.creation.TicketInput.View()

	case ReviewInput:
		view = "Select reviewer(s):\n\n"

		for i, choice := range m.creation.ReviewInput {

			if m.usr.Id == choice.Value {
				continue
			}

			cursor := " "
			if i == m.cursor {
				cursor = ">"
			}

			checked := "[ ]"
			if choice.Selected {
				checked = "[x]"
			}

			view += fmt.Sprintf("%s %s %s\n", cursor, checked, choice.Label)
		}

	case MergeRequestSummary:
		view = "Merge Request Summary:\n\n"

		// Display ticket number
		view += fmt.Sprintf("Ticket number: [Hapo-%s]\n\n", m.creation.TicketInput.Value())

		// Display title
		view += fmt.Sprintf("Title: %s\n\n", m.creation.TitleInput.Value())

		lines := strings.Split(m.creation.DescriptionInput.Value(), "\n")
		for i, line := range lines {
			lines[i] = "- " + line
		}
		formattedDescription := strings.Join(lines, "\n")

		// Display description
		view += "Description: \n"
		view += fmt.Sprintf("%s\n", formattedDescription)

		// Display selected reviewers
		view += "\nSelected Reviewers:\n"
		for _, choice := range m.creation.ReviewInput {
			if choice.Selected {
				view += fmt.Sprintf("- %s\n", choice.Label)
			}
		}

	case ErrorState:
		if m.err != nil {
			view = fmt.Sprintf("Something went wrong: %s\n", m.err.Error())
		}
	default:
		view = "Unknown state\n"

	}

	view += "\n\n" + m.helpText()

	return view
}

func filteredChoices(usr util.Author, allChoices []reviewInput) []reviewInput {
	var filteredChoices []reviewInput
	for _, choice := range allChoices {
		if choice.Value != usr.Id {
			filteredChoices = append(filteredChoices, choice)
		}
	}
	return filteredChoices
}

func (m Model) helpText() string {
	defaultText := "[ctrl+c] Quit"

	var additionalText string

	switch m.state {
	case DisplayTable:
		additionalText = "  [ctrl+y] Continue [esc] Focus/Defocus"

	case TitleInput:
		additionalText = "  [ctrl+y] Confirm Title"

	case DescriptionInput:
		additionalText = "  [ctrl+y] Confirm Description [Enter] Next line [Arrow Up/Down] Navigate (no vim :sad:)"

	case TicketInput:
		additionalText = " [ctrl+y] Confirm Ticket ID"

	case ReviewInput:
		additionalText = " [ctrl+y] Accept Reviewers [Enter] Select Option"

	case MergeRequestSummary:
		additionalText = " [ctrl+y] Create Merge Request"
	}

	return defaultText + additionalText
}
func checkServer() tea.Msg {
	author, err := util.CheckAuthUser()
	if err != nil {
		return errMsg{err}
	}

	return usr{Author: author}
}
