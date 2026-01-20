package autonomy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Actions struct {
	Version     string        `json:"version"`
	Summary     string        `json:"summary"`
	Next        ActionNext    `json:"next"`
	Updates     *ActionUpdate `json:"updates,omitempty"`
	CreateBeads []Task        `json:"create_beads,omitempty"`
	Gates       *ActionGates  `json:"gates,omitempty"`
	Stop        *ActionStop   `json:"stop,omitempty"`
}

type ActionNext struct {
	Type   string `json:"type"`
	ID     string `json:"id,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type ActionUpdate struct {
	Beads []ActionBeadUpdate `json:"beads,omitempty"`
}

type ActionBeadUpdate struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Note   string `json:"note,omitempty"`
}

type ActionGates struct {
	ReviewRequired bool     `json:"review_required"`
	Tests          []string `json:"tests,omitempty"`
	Evidence       []string `json:"evidence,omitempty"`
	Verification   []string `json:"verification,omitempty"`
	Blocking       bool     `json:"blocking"`
}

type ActionStop struct {
	Reason  string `json:"reason"`
	Details string `json:"details,omitempty"`
}

var (
	ErrActionsJSONMissing = errors.New("no actions json found")
	ErrActionsJSONInvalid = errors.New("invalid actions json")
)

func (c *Controller) actionsFromRun(runID string) (*Actions, error) {
	if strings.TrimSpace(c.Config.Autonomy.ActionsSchema) == "" {
		return nil, nil
	}
	artifacts, err := c.Store.ListArtifacts(runID)
	if err != nil {
		return nil, err
	}
	if len(artifacts) == 0 {
		return nil, nil
	}
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].CreatedAt.After(artifacts[j].CreatedAt)
	})

	var parseErr error
	for _, artifact := range artifacts {
		content, err := os.ReadFile(artifact.Path)
		if err != nil {
			continue
		}
		payload, err := extractJSONBlock(string(content))
		if err != nil {
			if errors.Is(err, ErrActionsJSONMissing) {
				continue
			}
			if c.Config.Autonomy.FailOnSchemaError != nil && !*c.Config.Autonomy.FailOnSchemaError {
				return nil, nil
			}
			parseErr = err
			continue
		}
		if err := validateJSONSchema(c.Config.Autonomy.ActionsSchema, payload); err != nil {
			if c.Config.Autonomy.FailOnSchemaError != nil && !*c.Config.Autonomy.FailOnSchemaError {
				return nil, nil
			}
			parseErr = fmt.Errorf("%w: schema validation failed: %v", ErrActionsJSONInvalid, err)
			continue
		}
		var actions Actions
		if err := json.Unmarshal([]byte(payload), &actions); err != nil {
			parseErr = fmt.Errorf("%w: parse error: %v", ErrActionsJSONInvalid, err)
			continue
		}
		return &actions, nil
	}
	if parseErr != nil {
		return nil, parseErr
	}
	return nil, nil
}

func (c *Controller) applyActions(beadID string, actions *Actions) (string, bool, bool, error) {
	stopReason := ""
	gateFailure := false
	updatedCurrent := false

	if actions == nil {
		return stopReason, gateFailure, updatedCurrent, nil
	}

	if actions.Stop != nil {
		stopReason = strings.TrimSpace(actions.Stop.Reason)
	}

	if actions.Gates != nil {
		if actions.Gates.Blocking || actions.Gates.ReviewRequired {
			gateFailure = true
		}
	}

	if c.Config.Beads.AutoUpdate {
		if !bdAvailable() {
			if c.Logger != nil {
				c.Logger.Warn("bd not found; skipping bead updates")
			}
		} else {
			if actions.Updates != nil {
				for _, update := range actions.Updates.Beads {
					id := sanitizeBeadID(update.ID)
					if id == "" {
						continue
					}
					if err := updateBeadStatus(id, update.Status); err != nil {
						return stopReason, gateFailure, updatedCurrent, err
					}
					if beadID != "" && id == beadID {
						updatedCurrent = true
					}
				}
			}

			if gateFailure && beadID != "" {
				status := "blocked"
				if actions.Gates != nil && actions.Gates.ReviewRequired {
					status = "review"
				}
				_ = updateBeadStatus(beadID, status)
			}
		}
	}

	if len(actions.CreateBeads) > 0 {
		if err := c.createBeads(TasksFile{Tasks: actions.CreateBeads}); err != nil {
			return stopReason, gateFailure, updatedCurrent, err
		}
	}

	return stopReason, gateFailure, updatedCurrent, nil
}

func extractJSONBlock(output string) (string, error) {
	const startMarker = "ACTIONS_JSON_START"
	const endMarker = "ACTIONS_JSON_END"

	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return "", ErrActionsJSONMissing
	}
	if idx := strings.Index(output, startMarker); idx != -1 {
		rest := output[idx+len(startMarker):]
		if end := strings.Index(rest, endMarker); end != -1 {
			payload := strings.TrimSpace(rest[:end])
			if json.Valid([]byte(payload)) {
				return payload, nil
			}
			return "", fmt.Errorf("%w: invalid json in markers", ErrActionsJSONInvalid)
		}
	}

	fences := strings.Split(output, "```")
	for i := 1; i < len(fences); i += 2 {
		block := strings.TrimSpace(fences[i])
		block = strings.TrimPrefix(block, "json")
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		if json.Valid([]byte(block)) {
			return block, nil
		}
		return "", fmt.Errorf("%w: invalid json in fenced block", ErrActionsJSONInvalid)
	}

	if json.Valid([]byte(trimmed)) {
		return trimmed, nil
	}
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return "", fmt.Errorf("%w: invalid json in output", ErrActionsJSONInvalid)
	}

	if start := strings.Index(trimmed, "{"); start != -1 {
		for end := len(trimmed); end > start; end-- {
			candidate := strings.TrimSpace(trimmed[start:end])
			if json.Valid([]byte(candidate)) {
				return candidate, nil
			}
		}
		return "", fmt.Errorf("%w: invalid json in output", ErrActionsJSONInvalid)
	}

	return "", ErrActionsJSONMissing
}
