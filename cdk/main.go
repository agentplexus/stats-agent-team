// CDK deployment for stats-agent-team
//
// This loads configuration from config.json and synthesizes CloudFormation.
//
// Prerequisites:
//  1. Create AWS secrets (see ROADMAP.md for secret paths)
//  2. Bootstrap CDK: cdk bootstrap aws://ACCOUNT/REGION
//
// Deploy with:
//
//	cd cdk
//	cdk deploy
package main

import (
	"github.com/agentplexus/agentkit-aws-cdk/agentcore"
)

func main() {
	app := agentcore.NewApp()

	// Load configuration from JSON file
	// Edit config.json to customize agents, secrets, and infrastructure
	agentcore.MustNewStackFromFile(app, "config.json")

	agentcore.Synth(app)
}
