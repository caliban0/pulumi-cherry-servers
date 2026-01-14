package integration_test

import (
	"os"
	"strconv"
	"testing"

	"github.com/caliban0/pulumi-cherry-servers/provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
)

func teamFromEnv(t *testing.T) int {
	t.Helper()

	const teamVar = "CHERRY_TEAM_ID"

	teamRaw, ok := os.LookupEnv(teamVar)
	if !ok {
		t.Fatalf("%s not set", teamVar)
	}

	team, err := strconv.Atoi(teamRaw)
	if err != nil {
		t.Fatalf("failed to parse %s: %v", teamVar, err)
	}
	return team
}

func TestProjectLifecycleWithOnlyRequiredArgs(t *testing.T) {
	server := newServer(t)

	team := teamFromEnv(t)

	integration.LifeCycleTest{
		Resource: provider.Name + ":provider:Project",
		Create: integration.Operation{
			Inputs: property.NewMap(map[string]property.Value{
				"team": property.New(float64(team)),
			}),
			Hook: func(_, output property.Map) {
				assert.Regexp(t, "test-([a-f]|[0-9]){6}", output.Get("name").AsString())
				assert.Equal(t, team, int(output.Get("team").AsNumber()))
				assert.False(t, output.Get("bgp").AsMap().Get("enabled").AsBool())
			},
		},
	}.Run(t, server)
}

func TestProjectLifecycleWithOptionalArgs(t *testing.T) {
	server := newServer(t)

	const name = "pulumi-test-project-optionals"
	team := teamFromEnv(t)

	integration.LifeCycleTest{
		Resource: provider.Name + ":provider:Project",
		Create: integration.Operation{
			Inputs: property.NewMap(map[string]property.Value{
				"name": property.New(name),
				"team": property.New(float64(team)),
				"bgp":  property.New(true),
			}),
			Hook: func(_, output property.Map) {
				assert.Equal(t, name, output.Get("name").AsString())
				assert.Equal(t, team, int(output.Get("team").AsNumber()))
				assert.True(t, output.Get("bgp").AsMap().Get("enabled").AsBool())
			},
		},
		Updates: []integration.Operation{
			{
				Inputs: property.NewMap(map[string]property.Value{
					"name": property.New(name + "-updated"),
					"team": property.New(float64(team)),
					"bgp":  property.New(false),
				}),
				Hook: func(_, output property.Map) {
					assert.Equal(t, name+"-updated", output.Get("name").AsString())
					assert.Equal(t, team, int(output.Get("team").AsNumber()))
					assert.False(t, output.Get("bgp").AsMap().Get("enabled").AsBool())
				},
			},
		},
	}.Run(t, server)
}
