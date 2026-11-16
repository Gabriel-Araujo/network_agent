package history

import "testing"

func TestProjectMermaidRepair(t *testing.T) {
	original := Message{ID: "answer", Role: RoleAgent, Status: StatusDone, Content: "original"}
	started := Message{Role: RoleRepair, MermaidRepair: &MermaidRepair{AttemptID: "attempt", MessageID: "answer", State: MermaidRepairStarted}}
	candidate := Message{Role: RoleRepair, MermaidRepair: &MermaidRepair{AttemptID: "attempt", MessageID: "answer", State: MermaidRepairCandidate}}
	accepted := Message{Role: RoleRepair, MermaidRepair: &MermaidRepair{AttemptID: "attempt", MessageID: "answer", State: MermaidRepairAccepted, CorrectedContent: "corrected"}}
	conv := Conversation{Messages: []Message{{ID: "question", Role: RoleUser}, original, started, candidate, accepted}}

	view := ProjectConversation(conv)
	if len(view.Messages) != 2 || view.Messages[1].Content != "original" || view.Messages[1].MermaidRepair == nil || view.Messages[1].MermaidRepair.CorrectedContent != "corrected" {
		t.Fatalf("view = %+v", view.Messages)
	}
	transcript := ProjectTranscript(conv.Messages)
	if len(transcript) != 2 || transcript[1].Content != "corrected" {
		t.Fatalf("transcript = %+v", transcript)
	}
	if conv.Messages[1].Content != "original" || len(conv.Messages) != 5 {
		t.Fatal("projection changed the stored conversation")
	}
}

func TestProjectMermaidRepairInterruptedOrRejected(t *testing.T) {
	for _, state := range []MermaidRepairState{MermaidRepairStarted, MermaidRepairCandidate, MermaidRepairRejected, MermaidRepairFailed} {
		t.Run(string(state), func(t *testing.T) {
			messages := []Message{
				{ID: "answer", Role: RoleAgent, Status: StatusDone, Content: "original"},
				{Role: RoleRepair, MermaidRepair: &MermaidRepair{AttemptID: "attempt", MessageID: "answer", State: state, CorrectedContent: "candidate"}},
			}
			view := ProjectConversation(Conversation{Messages: messages})
			if len(view.Messages) != 1 || view.Messages[0].MermaidRepair == nil {
				t.Fatalf("view = %+v", view.Messages)
			}
			if got := ProjectTranscript(messages); len(got) != 1 || got[0].Content != "original" {
				t.Fatalf("transcript = %+v", got)
			}
		})
	}
}

func TestProjectMermaidRepairIgnoresUnrelatedAttempt(t *testing.T) {
	messages := []Message{
		{ID: "answer", Role: RoleAgent, Content: "original"},
		{Role: RoleRepair, MermaidRepair: &MermaidRepair{AttemptID: "one", MessageID: "answer", State: MermaidRepairStarted}},
		{Role: RoleRepair, MermaidRepair: &MermaidRepair{AttemptID: "two", MessageID: "answer", State: MermaidRepairAccepted, CorrectedContent: "unrelated"}},
	}
	if got := ProjectTranscript(messages); len(got) != 1 || got[0].Content != "original" {
		t.Fatalf("transcript = %+v", got)
	}
}
