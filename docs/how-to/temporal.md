---
seotitle: How to use Temporal and Encore
seodesc: Learn how to use Temporal for reliable workflow execution with Encore.
title: Use Temporal with Encore
---

[Temporal](https://temporal.io) is a workflow orchestration system for building highly reliable systems.
Encore works great with Temporal, and this guide shows you how to integrate Temporal into your Encore application.

## Set up Temporal clusters
You'll need at least two Temporal clusters: one for local development and one for cloud environments.

We recommend using [Temporalite](https://github.com/temporalio/temporalite) for local development,
and [Temporal Cloud](https://temporal.io/cloud) for cloud environments. 

## Set up Temporal Workflow

Next it's time to create a Temporal Workflow. We'll base this on the Temporal [Hello World](https://learn.temporal.io/getting_started/go/hello_world_in_go/)
exampe.

Create a new Encore service named `greeting`:

```go
-- greeting/greeting.go --
package greeting

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"encore.dev"
)

// Use an environment-specific task queue so we can use the same
// Temporal Cluster for all cloud environments.
var (
    envName = encore.Meta().Environment.Name
    greetingTaskQueue = envName + "-greeting"
)

//encore:service
type Service struct {
	client client.Client
	worker worker.Worker
}

func initService() (*Service, error) {
	c, err := client.Dial(client.Options{})
	if err != nil {
		return nil, fmt.Errorf("create temporal client: %v", err)
	}

	w := worker.New(c, greetingTaskQueue, worker.Options{})

	err = w.Start()
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("start temporal worker: %v", err)
	}
	return &Service{client: c, worker: w}, nil
}

func (s *Service) Shutdown(force context.Context) {
	s.client.Close()
	s.worker.Stop()
}
```

Next it's time to define some workflows. These need to be in the same service,
so add a new `workflow` package inside the `greeting` service, containing
a workflow and activity definition in separate files:

```go
-- greeting/workflow/workflow.go --
package workflow

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

func Greeting(ctx workflow.Context, name string) (string, error) {
    options := workflow.ActivityOptions{
        StartToCloseTimeout: time.Second * 5,
    }

    ctx = workflow.WithActivityOptions(ctx, options)

    var result string
    err := workflow.ExecuteActivity(ctx, ComposeGreeting, name).Get(ctx, &result)

    return result, err
}
-- greeting/workflow/activity.go --
package workflow

import (
	"context"
	"fmt"
)

func ComposeGreeting(ctx context.Context, name string) (string, error) {
    greeting := fmt.Sprintf("Hello %s!", name)
    return greeting, nil
}
```

Then, go back to the `greeting` service and register the workflow and activity:

```go
-- greeting/greeting.go --
// Import the package at the top:
import "encore.app/greeting/workflow"

// Add these lines to `initService`, below the call to `worker.New`:
w.RegisterWorkflow(workflow.Greeting)
w.RegisterActivity(workflow.ComposeGreeting)
```

Now let's create an Encore API that triggers this workflow.

Add a new file `greeting/greet.go`:

```go
-- greeting/greet.go --
package greeting

import (
	"context"

	"encore.app/greeting/workflow"
	"encore.dev/rlog"
	"go.temporal.io/sdk/client"
)

type GreetResponse struct {
    Greeting string
}

//encore:api public path=/greet/:name
func (s *Service) Greet(ctx context.Context, name string) (*GreetResponse, error) {
    options := client.StartWorkflowOptions{
        ID:        "greeting-workflow",
        TaskQueue: greetingTaskQueue,
    }
    we, err := s.client.ExecuteWorkflow(ctx, options, workflow.Greeting, name)
    if err != nil {
        return nil, err
    }
    rlog.Info("started workflow", "id", we.GetID(), "run_id", we.GetRunID())

    // Get the results
    var greeting string
    err = we.Get(ctx, &greeting)
    if err != nil {
        return nil, err
    }
    return &GreetResponse{Greeting: greeting}, nil
}
```

## Run it locally

Now we're ready to test it out. Start up `temporalite` and your Encore application (in separate terminals):

```bash
$ temporalite start --namespace default
$ encore run
```

Now try calling it, either from the [Local Development Dashboard](/docs/observability/dev-dash) or using cURL:

```bash
$ curl 'http://localhost:4000/greeting/Temporal'
{"Greeting": "Hello Temporal!"}
```

If you see this, it works!

## Run in the cloud

To run it in the cloud, you will need to use Temporal Cloud or your own, self-hosted Temporal cluster.
The easiest way to automatically pick up the correct cluster address is to use Encore's [config functionality](/docs/develop/config).

Add two new files:
```
-- greeting/config.go --
package greeting

import "encore.dev/config"

type Config struct {
    TemporalServer string
}

var cfg = config.Load[*Config]()
-- greeting/config.cue --
package greeting

TemporalServer: [
	// These act as individual case statements
    if #Meta.Environment.Cloud == "local" { "localhost:7233" },

    // TODO: configure this to match your own cluster address
    "my.cluster.address:7233",
][0] // Return the first value which matches the condition
```

Finally go back to `greeting/greeting.go` and update the `client.Dial` call to look like:

```go
-- greeting/greeting.go --
client.Dial(client.Options{HostPort: cfg.TemporalServer})
```

With that, Encore will automatically connect to the correct Temporal cluster, using a local cluster
for local development and your cloud-hosted cluster for everything else.
