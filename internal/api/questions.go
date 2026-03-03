package api

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// TemplateCategory represents a category of questions in an audit template.
type TemplateCategory struct {
	CategoryName string                   `json:"categoryName"`
	Questions    []TemplateQuestion        `json:"questions"`
	Settings     TemplateCategorySettings  `json:"settings"`
}

// TemplateCategorySettings contains settings for a template category.
type TemplateCategorySettings struct {
	Duplicate bool `json:"duplicate"`
}

// TemplateQuestion represents a single question in a template category.
type TemplateQuestion struct {
	Question    string                   `json:"question"`
	Description string                   `json:"description"`
	Answer      []interface{}            `json:"answer"`
	Settings    TemplateQuestionSettings `json:"settings"`
	Ticket      []interface{}            `json:"ticket"`
}

// TemplateQuestionSettings contains settings for a template question.
type TemplateQuestionSettings struct {
	AnswerType     string                  `json:"answertype"`
	TicketRequired bool                    `json:"ticketRequired"`
	Choice         string                  `json:"choice,omitempty"`
	Answer         []string                `json:"answer,omitempty"`
	RichOptions    []RichOption            `json:"richOptions,omitempty"`
	Styling        *QuestionStyling        `json:"styling,omitempty"`
}

// RichOption represents an option in a multiplechoice question.
type RichOption struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Image string `json:"image"`
	Type  string `json:"type"`
}

// QuestionStyling contains styling options for a question.
type QuestionStyling struct {
	Options map[string]StylingOption `json:"options"`
}

// StylingOption represents the styling for a single answer option.
type StylingOption struct {
	BackgroundColor string `json:"backgroundColor"`
	Color           string `json:"color"`
	Label           string `json:"label"`
	SortIndex       int    `json:"sortIndex"`
}

var validAnswerTypes = map[string]bool{
	"yesnona":        true,
	"freetext":       true,
	"multiplechoice": true,
	"numeric":        true,
	"rating":         true,
	"date":           true,
	"time":           true,
	"duration":       true,
	"signature":      true,
	"statictext":     true,
}

// ValidateTemplateQuestions validates a slice of template categories.
// All errors are collected and returned as a single joined error.
func ValidateTemplateQuestions(categories []TemplateCategory) error {
	var errs []string

	if len(categories) == 0 {
		return fmt.Errorf("questions file must contain at least 1 category")
	}

	for ci, cat := range categories {
		catNum := ci + 1

		if strings.TrimSpace(cat.CategoryName) == "" {
			errs = append(errs, fmt.Sprintf("category %d: categoryName is required", catNum))
		}

		for qi, q := range cat.Questions {
			qNum := qi + 1
			prefix := fmt.Sprintf("category %d, question %d", catNum, qNum)

			if strings.TrimSpace(q.Question) == "" {
				errs = append(errs, fmt.Sprintf("%s: question text is required", prefix))
			}

			at := q.Settings.AnswerType
			if at == "" {
				errs = append(errs, fmt.Sprintf("%s: settings.answertype is required", prefix))
				continue
			}
			if !validAnswerTypes[at] {
				errs = append(errs, fmt.Sprintf("%s: invalid answertype %q (must be one of: yesnona, freetext, multiplechoice, numeric, rating, date, time, duration, signature, statictext)", prefix, at))
				continue
			}

			if at == "multiplechoice" {
				validateMultipleChoice(prefix, q.Settings, &errs)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation errors:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}

func validateMultipleChoice(prefix string, s TemplateQuestionSettings, errs *[]string) {
	if s.Choice != "single" && s.Choice != "multiple" {
		*errs = append(*errs, fmt.Sprintf("%s: multiplechoice question requires 'choice' field set to \"single\" or \"multiple\"", prefix))
	}

	if len(s.Answer) == 0 {
		*errs = append(*errs, fmt.Sprintf("%s: multiplechoice question requires at least 1 option ID in settings.answer", prefix))
	}

	if len(s.RichOptions) == 0 {
		*errs = append(*errs, fmt.Sprintf("%s: multiplechoice question requires at least 1 richOption", prefix))
		return
	}

	// Validate each richOption
	for i, ro := range s.RichOptions {
		roPrefix := fmt.Sprintf("%s, richOption %d", prefix, i+1)
		if ro.ID == "" {
			*errs = append(*errs, fmt.Sprintf("%s: id is required", roPrefix))
		}
		if strings.TrimSpace(ro.Text) == "" {
			*errs = append(*errs, fmt.Sprintf("%s: text is required", roPrefix))
		}
		if ro.Type != "textselect" {
			*errs = append(*errs, fmt.Sprintf("%s: type must be \"textselect\", got %q", roPrefix, ro.Type))
		}
	}

	// Cross-check: richOption IDs must match settings.answer IDs
	if len(s.Answer) > 0 && len(s.RichOptions) > 0 {
		answerIDs := make(map[string]bool)
		for _, id := range s.Answer {
			answerIDs[id] = true
		}
		richIDs := make(map[string]bool)
		for _, ro := range s.RichOptions {
			if ro.ID != "" {
				richIDs[ro.ID] = true
			}
		}

		for _, id := range s.Answer {
			if !richIDs[id] {
				*errs = append(*errs, fmt.Sprintf("%s: settings.answer references ID %q not found in richOptions", prefix, id))
			}
		}
		for _, ro := range s.RichOptions {
			if ro.ID != "" && !answerIDs[ro.ID] {
				*errs = append(*errs, fmt.Sprintf("%s: richOption ID %q not found in settings.answer", prefix, ro.ID))
			}
		}
	}
}

// LoadAndValidateQuestionsFile reads a JSON file, parses it as template categories,
// validates the structure, and returns the typed categories.
func LoadAndValidateQuestionsFile(path string) ([]TemplateCategory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading questions file: %w", err)
	}

	var categories []TemplateCategory
	if err := json.Unmarshal(data, &categories); err != nil {
		return nil, fmt.Errorf("parsing questions file: %w", err)
	}

	if err := ValidateTemplateQuestions(categories); err != nil {
		return nil, err
	}

	return categories, nil
}
