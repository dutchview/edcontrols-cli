package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateTemplateQuestions(t *testing.T) {
	tests := []struct {
		name    string
		input   []TemplateCategory
		wantErr string // empty means no error expected; substring match
	}{
		{
			name:    "empty categories",
			input:   []TemplateCategory{},
			wantErr: "at least 1 category",
		},
		{
			name: "valid minimal yesnona",
			input: []TemplateCategory{
				{
					CategoryName: "General",
					Settings:     TemplateCategorySettings{Duplicate: false},
					Questions: []TemplateQuestion{
						{
							Question: "<p>Is this ok?</p>",
							Settings: TemplateQuestionSettings{AnswerType: "yesnona"},
						},
					},
				},
			},
		},
		{
			name: "valid yesnona with styling",
			input: []TemplateCategory{
				{
					CategoryName: "Styled",
					Questions: []TemplateQuestion{
						{
							Question: "<p>Styled question</p>",
							Settings: TemplateQuestionSettings{
								AnswerType: "yesnona",
								Styling: &QuestionStyling{
									Options: map[string]StylingOption{
										"YES": {BackgroundColor: "#33cc66", Color: "#fff", Label: "Yes", SortIndex: 1},
										"NO":  {BackgroundColor: "#f84143", Color: "#fff", Label: "No", SortIndex: 2},
										"N/A": {BackgroundColor: "#9e9e9e", Color: "#fff", Label: "N/A", SortIndex: 3},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "valid multiplechoice single",
			input: []TemplateCategory{
				{
					CategoryName: "Choices",
					Questions: []TemplateQuestion{
						{
							Question: "<p>Pick one</p>",
							Settings: TemplateQuestionSettings{
								AnswerType: "multiplechoice",
								Choice:     "single",
								Answer:     []string{"O1", "O2"},
								RichOptions: []RichOption{
									{ID: "O1", Text: "<p>Option 1</p>", Type: "textselect"},
									{ID: "O2", Text: "<p>Option 2</p>", Type: "textselect"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "valid multiplechoice multiple",
			input: []TemplateCategory{
				{
					CategoryName: "Multi",
					Questions: []TemplateQuestion{
						{
							Question: "<p>Pick many</p>",
							Settings: TemplateQuestionSettings{
								AnswerType: "multiplechoice",
								Choice:     "multiple",
								Answer:     []string{"A", "B", "C"},
								RichOptions: []RichOption{
									{ID: "A", Text: "<p>A</p>", Type: "textselect"},
									{ID: "B", Text: "<p>B</p>", Type: "textselect"},
									{ID: "C", Text: "<p>C</p>", Type: "textselect"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "valid multiple answer types",
			input: []TemplateCategory{
				{
					CategoryName: "All Types",
					Questions: []TemplateQuestion{
						{Question: "Q1", Settings: TemplateQuestionSettings{AnswerType: "freetext"}},
						{Question: "Q2", Settings: TemplateQuestionSettings{AnswerType: "numeric"}},
						{Question: "Q3", Settings: TemplateQuestionSettings{AnswerType: "rating"}},
						{Question: "Q4", Settings: TemplateQuestionSettings{AnswerType: "date"}},
						{Question: "Q5", Settings: TemplateQuestionSettings{AnswerType: "time"}},
						{Question: "Q6", Settings: TemplateQuestionSettings{AnswerType: "duration"}},
						{Question: "Q7", Settings: TemplateQuestionSettings{AnswerType: "signature"}},
						{Question: "Q8", Settings: TemplateQuestionSettings{AnswerType: "statictext"}},
					},
				},
			},
		},
		{
			name: "missing categoryName",
			input: []TemplateCategory{
				{
					CategoryName: "",
					Questions: []TemplateQuestion{
						{Question: "Q", Settings: TemplateQuestionSettings{AnswerType: "freetext"}},
					},
				},
			},
			wantErr: "category 1: categoryName is required",
		},
		{
			name: "missing question text",
			input: []TemplateCategory{
				{
					CategoryName: "Cat",
					Questions: []TemplateQuestion{
						{Question: "", Settings: TemplateQuestionSettings{AnswerType: "freetext"}},
					},
				},
			},
			wantErr: "category 1, question 1: question text is required",
		},
		{
			name: "missing answertype",
			input: []TemplateCategory{
				{
					CategoryName: "Cat",
					Questions: []TemplateQuestion{
						{Question: "Q"},
					},
				},
			},
			wantErr: "category 1, question 1: settings.answertype is required",
		},
		{
			name: "invalid answertype",
			input: []TemplateCategory{
				{
					CategoryName: "Cat",
					Questions: []TemplateQuestion{
						{Question: "Q", Settings: TemplateQuestionSettings{AnswerType: "checkbox"}},
					},
				},
			},
			wantErr: `invalid answertype "checkbox"`,
		},
		{
			name: "multiplechoice missing choice",
			input: []TemplateCategory{
				{
					CategoryName: "Cat",
					Questions: []TemplateQuestion{
						{
							Question: "Q",
							Settings: TemplateQuestionSettings{
								AnswerType:  "multiplechoice",
								Answer:      []string{"O1"},
								RichOptions: []RichOption{{ID: "O1", Text: "A", Type: "textselect"}},
							},
						},
					},
				},
			},
			wantErr: "requires 'choice' field",
		},
		{
			name: "multiplechoice missing answer IDs",
			input: []TemplateCategory{
				{
					CategoryName: "Cat",
					Questions: []TemplateQuestion{
						{
							Question: "Q",
							Settings: TemplateQuestionSettings{
								AnswerType:  "multiplechoice",
								Choice:      "single",
								RichOptions: []RichOption{{ID: "O1", Text: "A", Type: "textselect"}},
							},
						},
					},
				},
			},
			wantErr: "at least 1 option ID in settings.answer",
		},
		{
			name: "multiplechoice missing richOptions",
			input: []TemplateCategory{
				{
					CategoryName: "Cat",
					Questions: []TemplateQuestion{
						{
							Question: "Q",
							Settings: TemplateQuestionSettings{
								AnswerType: "multiplechoice",
								Choice:     "single",
								Answer:     []string{"O1"},
							},
						},
					},
				},
			},
			wantErr: "at least 1 richOption",
		},
		{
			name: "multiplechoice richOption missing text",
			input: []TemplateCategory{
				{
					CategoryName: "Cat",
					Questions: []TemplateQuestion{
						{
							Question: "Q",
							Settings: TemplateQuestionSettings{
								AnswerType:  "multiplechoice",
								Choice:      "single",
								Answer:      []string{"O1"},
								RichOptions: []RichOption{{ID: "O1", Text: "", Type: "textselect"}},
							},
						},
					},
				},
			},
			wantErr: "richOption 1: text is required",
		},
		{
			name: "multiplechoice richOption wrong type",
			input: []TemplateCategory{
				{
					CategoryName: "Cat",
					Questions: []TemplateQuestion{
						{
							Question: "Q",
							Settings: TemplateQuestionSettings{
								AnswerType:  "multiplechoice",
								Choice:      "single",
								Answer:      []string{"O1"},
								RichOptions: []RichOption{{ID: "O1", Text: "A", Type: "image"}},
							},
						},
					},
				},
			},
			wantErr: `type must be "textselect"`,
		},
		{
			name: "multiplechoice ID mismatch - answer references missing richOption",
			input: []TemplateCategory{
				{
					CategoryName: "Cat",
					Questions: []TemplateQuestion{
						{
							Question: "Q",
							Settings: TemplateQuestionSettings{
								AnswerType:  "multiplechoice",
								Choice:      "single",
								Answer:      []string{"O1", "O2"},
								RichOptions: []RichOption{{ID: "O1", Text: "A", Type: "textselect"}},
							},
						},
					},
				},
			},
			wantErr: `settings.answer references ID "O2" not found in richOptions`,
		},
		{
			name: "multiplechoice ID mismatch - richOption not in answer",
			input: []TemplateCategory{
				{
					CategoryName: "Cat",
					Questions: []TemplateQuestion{
						{
							Question: "Q",
							Settings: TemplateQuestionSettings{
								AnswerType:  "multiplechoice",
								Choice:      "single",
								Answer:      []string{"O1"},
								RichOptions: []RichOption{
									{ID: "O1", Text: "A", Type: "textselect"},
									{ID: "O2", Text: "B", Type: "textselect"},
								},
							},
						},
					},
				},
			},
			wantErr: `richOption ID "O2" not found in settings.answer`,
		},
		{
			name: "multiple errors at once",
			input: []TemplateCategory{
				{
					CategoryName: "",
					Questions: []TemplateQuestion{
						{Question: "", Settings: TemplateQuestionSettings{AnswerType: "invalid"}},
						{Question: "Q", Settings: TemplateQuestionSettings{}},
					},
				},
			},
			wantErr: "categoryName is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemplateQuestions(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.wantErr)
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidateMultipleErrorsCollected(t *testing.T) {
	categories := []TemplateCategory{
		{
			CategoryName: "",
			Questions: []TemplateQuestion{
				{Question: "", Settings: TemplateQuestionSettings{}},
			},
		},
		{
			CategoryName: "Cat2",
			Questions: []TemplateQuestion{
				{Question: "Q", Settings: TemplateQuestionSettings{AnswerType: "bogus"}},
			},
		},
	}

	err := ValidateTemplateQuestions(categories)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	msg := err.Error()
	expected := []string{
		"category 1: categoryName is required",
		"category 1, question 1: question text is required",
		"category 1, question 1: settings.answertype is required",
		`invalid answertype "bogus"`,
	}
	for _, e := range expected {
		if !strings.Contains(msg, e) {
			t.Errorf("expected error to contain %q, got:\n%s", e, msg)
		}
	}
}

func TestLoadAndValidateQuestionsFile(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		categories := []TemplateCategory{
			{
				CategoryName: "Test",
				Settings:     TemplateCategorySettings{Duplicate: false},
				Questions: []TemplateQuestion{
					{
						Question:    "<p>Test question</p>",
						Description: "",
						Answer:      []interface{}{},
						Settings:    TemplateQuestionSettings{AnswerType: "yesnona", TicketRequired: false},
						Ticket:      []interface{}{},
					},
				},
			},
		}

		data, _ := json.Marshal(categories)
		path := filepath.Join(t.TempDir(), "questions.json")
		os.WriteFile(path, data, 0644)

		result, err := LoadAndValidateQuestionsFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 category, got %d", len(result))
		}
		if result[0].CategoryName != "Test" {
			t.Errorf("expected categoryName 'Test', got %q", result[0].CategoryName)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadAndValidateQuestionsFile("/nonexistent/file.json")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "reading questions file") {
			t.Errorf("expected file read error, got: %v", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.json")
		os.WriteFile(path, []byte(`{not json}`), 0644)

		_, err := LoadAndValidateQuestionsFile(path)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "parsing questions file") {
			t.Errorf("expected parse error, got: %v", err)
		}
	})

	t.Run("valid JSON but fails validation", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "empty.json")
		os.WriteFile(path, []byte(`[]`), 0644)

		_, err := LoadAndValidateQuestionsFile(path)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "at least 1 category") {
			t.Errorf("expected validation error, got: %v", err)
		}
	})
}
