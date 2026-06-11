package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/compliance-framework/agent/runner"
	"github.com/compliance-framework/agent/runner/proto"
	"github.com/compliance-framework/plugin-aws-eks/internal"
	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
)

type CompliancePlugin struct {
	logger     hclog.Logger
	config     *internal.PluginConfig
	policyData map[string]interface{}
}

func (l *CompliancePlugin) Configure(req *proto.ConfigureRequest) (*proto.ConfigureResponse, error) {
	parsedConfig, err := internal.ParseConfig(req.GetConfig())
	if err != nil {
		return nil, err
	}
	l.config = parsedConfig

	if req.GetPolicyData() != nil {
		l.policyData = req.GetPolicyData().AsMap()
	} else {
		l.policyData = nil
	}

	return &proto.ConfigureResponse{}, nil
}

func (l *CompliancePlugin) Init(req *proto.InitRequest, apiHelper runner.ApiHelper) (*proto.InitResponse, error) {
	ctx := context.Background()
	return runner.InitWithSubjectsAndRisksFromPolicies(ctx, l.logger, req, apiHelper, buildSubjectTemplates())
}

func (l *CompliancePlugin) Eval(request *proto.EvalRequest, apiHelper runner.ApiHelper) (*proto.EvalResponse, error) {
	if request == nil {
		return &proto.EvalResponse{Status: proto.ExecutionStatus_FAILURE}, fmt.Errorf("eval request is nil")
	}

	ctx := context.Background()
	evalStatus := proto.ExecutionStatus_SUCCESS
	var accumulatedErrors error

	policyEval := request.WithDefaultPolicyBehavior(defaultPolicyBehaviorMapping())
	policyPathsByBehavior := buildPolicyPathsByBehavior(policyEval)
	requiredDatasets := buildRequiredDatasets(policyPathsByBehavior)
	if len(requiredDatasets) == 0 {
		l.logger.Info("No EKS policy behaviors matched policy paths")
		return &proto.EvalResponse{Status: proto.ExecutionStatus_SUCCESS}, nil
	}

	deps := internal.EvaluationDependencies{
		Context:      ctx,
		Logger:       l.logger,
		ApiHelper:    apiHelper,
		Actors:       buildOriginActors(),
		PolicyData:   l.policyData,
		PolicyLabels: policyLabels(l.config),
	}

	for _, region := range internal.ResolveRegions(l.config) {
		l.logger.Info("Collecting EKS resources in region", "region", region)

		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
		if err != nil {
			l.logger.Error("unable to load AWS SDK config for region", "region", region, "error", err)
			evalStatus = proto.ExecutionStatus_FAILURE
			accumulatedErrors = errors.Join(accumulatedErrors, err)
			continue
		}

		client := eks.NewFromConfig(cfg)
		datasets, err := internal.CollectRegionDatasets(ctx, l.logger, client, requiredDatasets)
		if err != nil {
			evalStatus = proto.ExecutionStatus_FAILURE
			accumulatedErrors = errors.Join(accumulatedErrors, err)
			continue
		}

		if clusterPolicyPaths := policyPathsByBehavior["cluster"]; len(clusterPolicyPaths) > 0 {
			result := internal.EvaluateClusterPolicies(deps, clusterPolicyPaths, datasets.Clusters, region, datasets)
			if fatal := applyResourceEvaluationErrors(result, &evalStatus, &accumulatedErrors); fatal != nil {
				return &proto.EvalResponse{Status: proto.ExecutionStatus_FAILURE}, fatal
			}
		}

		if nodegroupPolicyPaths := policyPathsByBehavior["nodegroup"]; len(nodegroupPolicyPaths) > 0 {
			result := internal.EvaluateNodegroupPolicies(deps, nodegroupPolicyPaths, datasets.Nodegroups, region, datasets)
			if fatal := applyResourceEvaluationErrors(result, &evalStatus, &accumulatedErrors); fatal != nil {
				return &proto.EvalResponse{Status: proto.ExecutionStatus_FAILURE}, fatal
			}
		}

		if addonPolicyPaths := policyPathsByBehavior["addon"]; len(addonPolicyPaths) > 0 {
			result := internal.EvaluateAddonPolicies(deps, addonPolicyPaths, datasets.Addons, region, datasets)
			if fatal := applyResourceEvaluationErrors(result, &evalStatus, &accumulatedErrors); fatal != nil {
				return &proto.EvalResponse{Status: proto.ExecutionStatus_FAILURE}, fatal
			}
		}
	}

	return &proto.EvalResponse{Status: evalStatus}, accumulatedErrors
}

func applyResourceEvaluationErrors(result internal.ResourceEvaluationErrors, evalStatus *proto.ExecutionStatus, accumulatedErrors *error) error {
	if result.NonFatal != nil {
		*accumulatedErrors = errors.Join(*accumulatedErrors, result.NonFatal)
		*evalStatus = proto.ExecutionStatus_FAILURE
	}
	if result.InputBuildFailure {
		*evalStatus = proto.ExecutionStatus_FAILURE
	}
	return result.Fatal
}

func defaultPolicyBehaviorMapping() map[string][]string {
	return map[string][]string{
		"aws-eks-policies":           {"cluster"},
		"aws-eks-nodegroup-policies": {"nodegroup"},
		"aws-eks-addon-policies":     {"addon"},
	}
}

func supportedPolicyBehaviors() []string {
	return []string{
		"cluster",
		"nodegroup",
		"addon",
	}
}

func buildPolicyPathsByBehavior(request *proto.EvalRequest) map[string][]string {
	policyPathsByBehavior := make(map[string][]string)
	for _, behavior := range supportedPolicyBehaviors() {
		policyPaths := request.PolicyPathsForBehavior(behavior)
		if len(policyPaths) > 0 {
			policyPathsByBehavior[behavior] = policyPaths
		}
	}
	return policyPathsByBehavior
}

func buildRequiredDatasets(policyPathsByBehavior map[string][]string) map[string]bool {
	requiredDatasets := make(map[string]bool)
	for behavior, policyPaths := range policyPathsByBehavior {
		if len(policyPaths) == 0 {
			continue
		}

		switch behavior {
		case "cluster":
			markRequiredDatasets(requiredDatasets, "clusters", "addons")
		case "nodegroup":
			markRequiredDatasets(requiredDatasets, "clusters", "nodegroups")
		case "addon":
			markRequiredDatasets(requiredDatasets, "clusters", "addons")
		}
	}
	return requiredDatasets
}

func markRequiredDatasets(requiredDatasets map[string]bool, datasetNames ...string) {
	for _, datasetName := range datasetNames {
		requiredDatasets[datasetName] = true
	}
}

func policyLabels(config *internal.PluginConfig) map[string]string {
	if config == nil {
		return nil
	}
	return config.PolicyLabels
}

func buildOriginActors() []*proto.OriginActor {
	return []*proto.OriginActor{
		{
			Title: "The Continuous Compliance Framework",
			Type:  "assessment-platform",
			Links: []*proto.Link{
				{
					Href: "https://compliance-framework.github.io/docs/",
					Rel:  internal.StringAddressed("reference"),
					Text: internal.StringAddressed("The Continuous Compliance Framework"),
				},
			},
		},
		{
			Title: "Continuous Compliance Framework - AWS EKS Plugin",
			Type:  "tool",
			Links: []*proto.Link{
				{
					Href: "https://github.com/compliance-framework/plugin-aws-eks",
					Rel:  internal.StringAddressed("reference"),
					Text: internal.StringAddressed("The Continuous Compliance Framework AWS EKS Plugin"),
				},
			},
		},
	}
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		JSONFormat: true,
	})

	compliancePluginObj := &CompliancePlugin{
		logger: logger,
	}
	logger.Debug("Initiating AWS EKS plugin")

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: runner.HandshakeConfig,
		Plugins: map[string]goplugin.Plugin{
			"runner": &runner.RunnerV2GRPCPlugin{
				Impl: compliancePluginObj,
			},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})
}
