// internal/ui/model_test.go
package ui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"hm/internal/ui"
)

func TestModel_ViewContainsCommand(t *testing.T) {
	m := ui.New("kubectl get pods -A", nil)
	view := m.View()
	if !strings.Contains(view, "kubectl get pods -A") {
		t.Errorf("view does not contain command:\n%s", view)
	}
}

func TestModel_ViewContainsHints(t *testing.T) {
	m := ui.New("ls", nil)
	view := m.View()
	if !strings.Contains(view, "[enter]") {
		t.Errorf("view missing [enter] hint:\n%s", view)
	}
	if !strings.Contains(view, "[e]") {
		t.Errorf("view missing [e] hint:\n%s", view)
	}
}

func TestModel_EscKey_Quits(t *testing.T) {
	m := ui.New("ls", nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if msg := cmd(); msg == nil {
		t.Fatal("Esc should return a quit command")
	} else if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("Esc key: expected QuitMsg, got %T", msg)
	}
}

func TestModel_QKey_Quits(t *testing.T) {
	m := ui.New("ls", nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("q key should produce QuitMsg")
	}
}

func TestModel_NKey_Quits(t *testing.T) {
	m := ui.New("ls", nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("n key should produce QuitMsg")
	}
}

func TestModel_EnterKey_SetsShouldCopyAndQuits(t *testing.T) {
	m := ui.New("kubectl get pods", nil)
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um := newM.(ui.Model)
	if !um.ShouldCopy() {
		t.Error("Enter should set ShouldCopy = true")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("Enter should produce QuitMsg")
	}
}

func TestModel_EnterKey_EmptyCommand_NoOp(t *testing.T) {
	m := ui.New("", nil)
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um := newM.(ui.Model)
	if um.ShouldCopy() {
		t.Error("Enter on empty command should not set ShouldCopy")
	}
}

func TestModel_EKey_EntersRefineMode(t *testing.T) {
	m := ui.New("ls", nil)
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	um := newM.(ui.Model)
	if !um.IsRefining() {
		t.Error("e key should enter refine mode")
	}
}

func TestModel_EscInRefineMode_ReturnsToNormal(t *testing.T) {
	m := ui.New("ls", nil)
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	newM2, _ := newM.(ui.Model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	um := newM2.(ui.Model)
	if um.IsRefining() {
		t.Error("Esc in refine mode should return to normal")
	}
}

func TestModel_EmptyCommand_ViewShowsPlaceholder(t *testing.T) {
	m := ui.New("", nil)
	view := m.View()
	if !strings.Contains(view, "No command returned") {
		t.Errorf("empty command should show placeholder:\n%s", view)
	}
}

func TestModel_AskResult_UpdatesCommand(t *testing.T) {
	m := ui.New("kubectl get pods -A", nil)

	// Enter refine mode and type a follow-up
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	um := newM.(ui.Model)
	for _, r := range "change namespace to my-ns" {
		newM, _ = um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		um = newM.(ui.Model)
	}
	// Simulate the AskResultMsg that the async Cmd would deliver
	newM, _ = um.Update(ui.AskResultMsg{Command: "kubectl get pods -n my-ns", SessionID: "s2"})
	um = newM.(ui.Model)

	if um.Command() != "kubectl get pods -n my-ns" {
		t.Errorf("Command after refine = %q", um.Command())
	}
}
