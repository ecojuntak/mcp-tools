package mcptools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-github/v60/github"
	"github.com/shaharia-lab/goai"
)

// GetRepositoryTool returns a tool for managing GitHub repositories
func (g *GitHub) GetRepositoryTool() goai.Tool {
	return goai.Tool{
		Name:        GitHubRepositoryToolName,
		Description: "Manages GitHub repositories - create, delete, update, fork",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"operation": {
					"type": "string",
					"enum": ["create", "delete", "update", "fork", "list_branches", "create_branch", "protect_branch"],
					"description": "Repository operation to perform"
				},
				"owner": {
					"type": "string",
					"description": "Repository owner"
				},
				"repo": {
					"type": "string",
					"description": "Repository name"
				},
				"description": {
					"type": "string",
					"description": "Repository description"
				},
				"private": {
					"type": "boolean",
					"description": "Whether the repository should be private"
				},
				"branch": {
					"type": "string",
					"description": "Branch name for branch operations"
				},
				"source_branch": {
					"type": "string",
					"description": "Source branch for new branch creation"
				}
			},
			"required": ["operation"]
		}`),
		Handler: g.handleRepositoryOperation,
	}
}

func (g *GitHub) handleRepositoryOperation(ctx context.Context, params goai.CallToolParams) (goai.CallToolResult, error) {
	ctx, span := goai.StartSpan(ctx, fmt.Sprintf("%s.Handler", params.Name))
	defer span.End()

	var input struct {
		Operation    string `json:"operation"`
		Owner        string `json:"owner"`
		Repo         string `json:"repo"`
		Description  string `json:"description"`
		Private      bool   `json:"private"`
		Branch       string `json:"branch"`
		SourceBranch string `json:"source_branch"`
	}

	g.logger.WithFields(map[string]interface{}{
		"tool":      params.Name,
		"operation": params.Arguments,
	}).Info("handling repository operation")

	if err := json.Unmarshal(params.Arguments, &input); err != nil {
		return goai.CallToolResult{}, fmt.Errorf("failed to unmarshal input: %w", err)
	}

	var result interface{}
	var err error

	switch input.Operation {
	case "create":
		result, _, err = g.client.Repositories.Create(ctx, "", &github.Repository{
			Name:        &input.Repo,
			Description: &input.Description,
			Private:     &input.Private,
		})
	case "delete":
		_, err = g.client.Repositories.Delete(ctx, input.Owner, input.Repo)
		if err == nil {
			result = map[string]string{"status": "deleted"}
		}
	case "update":
		result, _, err = g.client.Repositories.Edit(ctx, input.Owner, input.Repo, &github.Repository{
			Description: &input.Description,
			Private:     &input.Private,
		})
	case "fork":
		result, _, err = g.client.Repositories.CreateFork(ctx, input.Owner, input.Repo, &github.RepositoryCreateForkOptions{})
	case "list_branches":
		result, _, err = g.client.Repositories.ListBranches(ctx, input.Owner, input.Repo, &github.BranchListOptions{})
	case "create_branch":
		// Get the source branch's SHA
		ref, _, err := g.client.Git.GetRef(ctx, input.Owner, input.Repo, "refs/heads/"+input.SourceBranch)
		if err != nil {
			return goai.CallToolResult{}, err
		}
		result, _, _ = g.client.Git.CreateRef(ctx, input.Owner, input.Repo, &github.Reference{
			Ref: github.String("refs/heads/" + input.Branch),
			Object: &github.GitObject{
				SHA: ref.Object.SHA,
			},
		})
	case "protect_branch":
		result, _, err = g.client.Repositories.UpdateBranchProtection(ctx, input.Owner, input.Repo, input.Branch,
			&github.ProtectionRequest{
				RequiredStatusChecks: &github.RequiredStatusChecks{
					Strict: true,
				},
				RequiredPullRequestReviews: &github.PullRequestReviewsEnforcementRequest{
					RequiredApprovingReviewCount: 1,
				},
			})
	default:
		return returnErrorOutput(fmt.Errorf("unsupported operation: %s", input.Operation)), nil
	}

	if err != nil {
		g.logger.WithFields(map[string]interface{}{
			"tool":                      params.Name,
			goai.ErrorLogField: err,
			"operation":                 input.Operation,
		}).Error("GitHub repository operation failed")

		return returnErrorOutput(fmt.Errorf("github repository %s error: %w", input.Operation, err)), nil
	}

	m := mustMarshal(result)
	g.logger.WithFields(map[string]interface{}{
		"tool":          params.Name,
		"operation":     input.Operation,
		"result_length": len(m),
	}).Info("GitHub repository operation completed successfully")

	return goai.CallToolResult{
		Content: []goai.ToolResultContent{{
			Type: "json",
			Text: m,
		}},
	}, nil
}
