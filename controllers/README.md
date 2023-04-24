# Controllers

## Plant Controller

Plant controller relies on the assumption that resources which are required to be generated (assuming, many)
should have a simple and straightforward way to be implemented and integrated into the chain.

That is why the internal logic of `PlantReconciler` depends of `workflow.Manager` which
manages the actual business logic, and the `StateHandler`(not yet an interface),
which bridges between the dependencies and control loop states.
Workflow Manager (and similar components) can be used to define the actual business logic required 
for creating, managing, and removing the API resource-based logic. 

Check documentation and actual usage.

```golang
type PlantReconciler struct {
    Client   client.Client
    Scheme   *runtime.Scheme
    Workflow workflow.Manager // separates business from state-control logic
    Recorder record.EventRecorder
}

type AbstractedReconciler interface {
    // Just an initializer
    Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
    
    // StateHandler
    ErrorHandle(ctx context.Context, plant *v1.Plant, err error) (ctrl.Result, error)
    StateHandle(ctx context.Context, plant *v1.Plant) (bool, error)
    UpdateStatus(ctx context.Context, plant *v1.Plant, opts ...func (*v1.Plant)) error
	
    // Business logic (minimal)
    HandleProcessingState(ctx context.Context, plant *v1.Plant) (bool, error)
    HandleDeletingState(ctx context.Context, plant *v1.Plant) (bool, error)
}

// This is an owner/parent resource, consider extending
type Plant struct {
	State string // e.g., State field which can be used for state-based control
	
	// everything else 
}
```
