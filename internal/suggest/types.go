package suggest

import "go-database/internal/plugin"

type SuggestionType string

const (
	SuggKeyword   SuggestionType = "keyword"
	SuggTable     SuggestionType = "table"
	SuggColumn    SuggestionType = "column"
	SuggStatement SuggestionType = "statement"
	SuggIntent    SuggestionType = "intent"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "LOW"
	RiskMedium RiskLevel = "MEDIUM"
	RiskHigh   RiskLevel = "HIGH"
)

type Suggestion struct {
	Text        string         `json:"text"`
	Description string         `json:"description"`
	Type        SuggestionType `json:"type"`
	Confidence  float64        `json:"confidence"`
	RiskLevel   RiskLevel      `json:"risk_level"`
}

type Context struct {
	UserID       string         `json:"user_id"`
	Role         string         `json:"role"`
	ConnectionID string         `json:"connection_id"`
	CurrentTable string         `json:"current_table"`
	Input        string         `json:"input"`
	Schema       *plugin.Schema `json:"-"`
}

type CommandType string

const (
	CmdSelect   CommandType = "SELECT"
	CmdInsert   CommandType = "INSERT"
	CmdUpdate   CommandType = "UPDATE"
	CmdDelete   CommandType = "DELETE"
	CmdCreate   CommandType = "CREATE"
	CmdDrop     CommandType = "DROP"
	CmdAlter    CommandType = "ALTER"
	CmdTruncate CommandType = "TRUNCATE"
	CmdUnknown  CommandType = "UNKNOWN"
)
