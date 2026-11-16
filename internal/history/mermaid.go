package history

type MermaidRepairState string

const (
	MermaidRepairStarted   MermaidRepairState = "started"
	MermaidRepairCandidate MermaidRepairState = "candidate"
	MermaidRepairAccepted  MermaidRepairState = "accepted"
	MermaidRepairRejected  MermaidRepairState = "rejected"
	MermaidRepairFailed    MermaidRepairState = "failed"
)

type MermaidFailure struct {
	Index  int    `json:"index"`
	Source string `json:"source"`
	Error  string `json:"error"`
}

type MermaidReplacement struct {
	Index  int    `json:"index"`
	Source string `json:"source"`
}

// MermaidRepair is an append-only event tied to an original message and attempt.
type MermaidRepair struct {
	AttemptID        string               `json:"attemptId"`
	MessageID        string               `json:"messageId"`
	State            MermaidRepairState   `json:"state"`
	Failures         []MermaidFailure     `json:"failures,omitempty"`
	Replacements     []MermaidReplacement `json:"replacements,omitempty"`
	CorrectedContent string               `json:"correctedContent,omitempty"`
	Error            string               `json:"error,omitempty"`
	Usage            *TokenUsage          `json:"usage,omitempty"`
}

// ProjectConversation hides repair events while retaining the original content.
func ProjectConversation(c Conversation) Conversation {
	messages := make([]Message, 0, len(c.Messages))
	positions := make(map[string]int)
	for _, msg := range c.Messages {
		if msg.Role != RoleRepair {
			positions[msg.ID] = len(messages)
			messages = append(messages, msg)
			continue
		}
		repair := msg.MermaidRepair
		if repair == nil || repair.AttemptID == "" {
			continue
		}
		pos, found := positions[repair.MessageID]
		if !found || messages[pos].Role != RoleAgent {
			continue
		}
		previous := messages[pos].MermaidRepair
		if previous != nil && previous.AttemptID != repair.AttemptID {
			continue
		}
		// A terminal result must be tied to a recorded attempt, not a stray event.
		if repair.State == MermaidRepairAccepted && (previous == nil || previous.State != MermaidRepairCandidate) {
			continue
		}
		copy := *repair
		messages[pos].MermaidRepair = &copy
	}
	c.Messages = messages
	return c
}

// ProjectTranscript substitutes only client-confirmed repairs for future turns.
func ProjectTranscript(messages []Message) []Message {
	view := ProjectConversation(Conversation{Messages: messages})
	for i := range view.Messages {
		repair := view.Messages[i].MermaidRepair
		if repair != nil && repair.State == MermaidRepairAccepted && repair.CorrectedContent != "" {
			view.Messages[i].Content = repair.CorrectedContent
		}
	}
	return view.Messages
}
