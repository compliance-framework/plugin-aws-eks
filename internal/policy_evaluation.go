package internal

import (
	"context"
	"errors"
	"strings"

	policyManager "github.com/compliance-framework/agent/policy-manager"
	"github.com/compliance-framework/agent/runner"
	"github.com/compliance-framework/agent/runner/proto"
	"github.com/hashicorp/go-hclog"
)

type EvaluationDependencies struct {
	Context      context.Context
	Logger       hclog.Logger
	ApiHelper    runner.ApiHelper
	Actors       []*proto.OriginActor
	PolicyData   map[string]interface{}
	PolicyLabels map[string]string
}

type ResourceEvidenceContext struct {
	Labels     map[string]string
	Components []*proto.Component
	Inventory  []*proto.InventoryItem
	Subjects   []*proto.Subject
}

type ResourceEvaluationErrors struct {
	Fatal             error
	NonFatal          error
	InputBuildFailure bool
}

func EvaluateResources[T any](deps EvaluationDependencies, policyPaths []string, resources []T, buildContext func(T) ResourceEvidenceContext, buildInput func(T) (interface{}, error), onInputError func(T, error), afterGenerate func([]*proto.Evidence, T)) ResourceEvaluationErrors {
	var accumulatedErrors error
	inputBuildFailure := false

	for _, resource := range resources {
		resourceCtx := buildContext(resource)
		input, err := buildInput(resource)
		if err != nil {
			inputBuildFailure = true
			if onInputError != nil {
				onInputError(resource, err)
			}
			accumulatedErrors = errors.Join(accumulatedErrors, err)
			continue
		}

		labels := MergeMaps(deps.PolicyLabels, resourceCtx.Labels)
		evidences, err := GenerateResourceEvidences(deps.Context, deps.Logger, deps.Actors, policyPaths, input, labels, resourceCtx.Subjects, resourceCtx.Components, resourceCtx.Inventory, deps.PolicyData)
		if afterGenerate != nil {
			afterGenerate(evidences, resource)
		}
		if err != nil {
			accumulatedErrors = errors.Join(accumulatedErrors, err)
		}
		if len(evidences) == 0 {
			continue
		}
		if err = deps.ApiHelper.CreateEvidence(deps.Context, evidences); err != nil {
			deps.Logger.Error("Failed to send evidences", "error", err)
			return ResourceEvaluationErrors{Fatal: err, NonFatal: accumulatedErrors, InputBuildFailure: inputBuildFailure}
		}
	}

	return ResourceEvaluationErrors{NonFatal: accumulatedErrors, InputBuildFailure: inputBuildFailure}
}

func NewResourceEvidenceContext(labels map[string]string, subjects []*proto.Subject, components []*proto.Component, inventory []*proto.InventoryItem) ResourceEvidenceContext {
	return ResourceEvidenceContext{
		Labels:     labels,
		Components: components,
		Inventory:  inventory,
		Subjects:   subjects,
	}
}

func GenerateResourceEvidences(ctx context.Context, logger hclog.Logger, actors []*proto.OriginActor, policyPaths []string, input interface{}, labels map[string]string, subjects []*proto.Subject, components []*proto.Component, inventory []*proto.InventoryItem, policyData map[string]interface{}) ([]*proto.Evidence, error) {
	activities := make([]*proto.Activity, 0)
	evidences := make([]*proto.Evidence, 0)
	var accumulatedErrors error

	for _, policyPath := range policyPaths {
		processor := policyManager.NewPolicyProcessor(
			logger,
			labels,
			subjects,
			components,
			inventory,
			actors,
			activities,
			policyData,
		)
		evidence, err := processor.GenerateResults(ctx, policyPath, input)
		evidences = append(evidences, evidence...)
		if err != nil {
			accumulatedErrors = errors.Join(accumulatedErrors, err)
		}
	}

	return evidences, accumulatedErrors
}

func PrefixEvidenceTitles(evidences []*proto.Evidence, prefix string) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return
	}

	for _, evidence := range evidences {
		if evidence == nil {
			continue
		}
		title := strings.TrimSpace(evidence.GetTitle())
		if title == "" {
			evidence.Title = prefix
			continue
		}
		evidence.Title = prefix + " | " + title
	}
}
