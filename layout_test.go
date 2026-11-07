package main

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/resendlabs/resend-go"
)

func TestShortTerminalShowsAddressAndSubjectFields(t *testing.T) {
	m := NewModel(resend.SendEmailRequest{}, SMTP)
	originalBodyHeight := m.Body.Height()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 102, Height: 13})
	m = updated.(Model)
	view := m.View()
	for _, field := range []string{"From", "me@example.com", "To", "you@example.com", "Subject", "Hello!"} {
		if !strings.Contains(ansi.Strip(view.Content), field) {
			t.Errorf("13-row composer is missing %q", field)
		}
	}
	if lipgloss.Height(view.Content) > 13 {
		t.Fatal("composer exceeds terminal height")
	}
	if m.Body.Height() >= originalBodyHeight {
		t.Fatal("body must shrink to make room for the fields")
	}
}

func TestShortTerminalKeepsFocusedControlVisible(t *testing.T) {
	for _, height := range []int{1, 5, 13, 40} {
		for _, state := range []State{editingFrom, editingTo, editingCc, editingBcc, editingSubject, editingBody, editingAttachments, hoveringSendButton} {
			t.Run(fmt.Sprintf("height=%d/state=%d", height, state), func(t *testing.T) {
				m := NewModel(resend.SendEmailRequest{Cc: []string{"cc@example.test"}}, SMTP)
				updated, _ := m.Update(tea.WindowSizeMsg{Width: 102, Height: height})
				m = updated.(Model)
				m.blurInputs()
				m.state = state
				m.focusActiveInput()
				view := m.View()
				if lipgloss.Height(view.Content) > height {
					t.Fatal("view exceeds terminal height")
				}
				if state <= editingBody {
					if view.Cursor == nil || view.Cursor.Y < 0 || view.Cursor.Y >= height {
						t.Fatalf("focused cursor outside view: %+v", view.Cursor)
					}
				}
				content := ansi.Strip(view.Content)
				if state == editingAttachments && !strings.Contains(content, "No attachments.") {
					t.Fatalf("attachment selection is not visible: %q", content)
				}
				if state == hoveringSendButton && !strings.Contains(content, "Send") {
					t.Fatalf("send button is not visible: %q", content)
				}
			})
		}
	}
}

func TestResizePreservesDraftAndRestoresBodyHeight(t *testing.T) {
	m := NewModel(resend.SendEmailRequest{From: "sender@example.test", To: []string{"recipient@example.test"}, Subject: "Draft", Text: "first line\nsecond line"}, SMTP)
	originalBodyHeight := m.Body.Height()
	m.blurInputs()
	m.state = editingBody
	m.focusActiveInput()
	m.Body.CursorStart()
	line, column := m.Body.Line(), m.Body.LineInfo().ColumnOffset
	for _, size := range []tea.WindowSizeMsg{{Width: 102, Height: 13}, {Width: 102, Height: 40}, {Width: 2, Height: 5}, {Width: 102, Height: 40}} {
		updated, _ := m.Update(size)
		m = updated.(Model)
		_ = m.View()
		if m.From.Value() != "sender@example.test" || m.To.Value() != "recipient@example.test" || m.Subject.Value() != "Draft" || m.Body.Value() != "first line\nsecond line" {
			t.Fatal("resize changed draft")
		}
		if !m.Body.Focused() || m.state != editingBody || m.Body.Line() != line {
			t.Fatal("resize changed editing state")
		}
	}
	if m.Body.Height() != originalBodyHeight {
		t.Fatal("growing terminal did not restore body height")
	}
	if m.Body.LineInfo().ColumnOffset != column {
		t.Fatal("resize moved body cursor")
	}
}

func TestShortTerminalKeepsSelectedAttachmentVisible(t *testing.T) {
	for _, height := range []int{1, 5, 13} {
		for _, selected := range []int{0, 5, 11} {
			t.Run(fmt.Sprintf("height=%d/selected=%d", height, selected), func(t *testing.T) {
				m := NewModel(resend.SendEmailRequest{}, SMTP)
				for i := range 12 {
					m.Attachments.InsertItem(i, attachment(fmt.Sprintf("/tmp/file-%02d.txt", i)))
				}
				m.Attachments.SetHeight(len(m.Attachments.Items()) + 2)
				m.Attachments.Select(selected)
				updated, _ := m.Update(tea.WindowSizeMsg{Width: 102, Height: height})
				m = updated.(Model)
				m.blurInputs()
				m.state = editingAttachments
				m.focusActiveInput()
				content := ansi.Strip(m.View().Content)
				if !strings.Contains(content, fmt.Sprintf("• file-%02d.txt", selected)) {
					t.Fatalf("selected attachment hidden: %q", content)
				}
			})
		}
	}
}

func TestShortTerminalTabNavigation(t *testing.T) {
	m := NewModel(resend.SendEmailRequest{}, SMTP)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 102, Height: 5})
	m = updated.(Model)
	for _, state := range []State{editingFrom, editingTo, editingSubject, editingBody, editingAttachments, hoveringSendButton, editingFrom} {
		if m.state != state {
			t.Fatalf("expected state %d, got %d", state, m.state)
		}
		view := m.View()
		if lipgloss.Height(view.Content) > 5 {
			t.Fatal("view exceeds terminal")
		}
		if state <= editingBody && (view.Cursor == nil || view.Cursor.Y < 0 || view.Cursor.Y >= 5) {
			t.Fatalf("focused input is off screen: %+v", view.Cursor)
		}
		updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		m = updated.(Model)
	}
}
