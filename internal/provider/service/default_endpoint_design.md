## CREATE

If the default endpoint is specified in the resource configuration, the provider will create a default endpoint for the service.
And save the data in the default_endpoint attribute.
Create, also calls update


## UPDATE

If the default endpoint is not specified in the resource configuration, but it is in the state, then the default endpoint will NOT be deleted. But it will be removed from the state.
If the default endpoint is specified in the resource configuration, the provider will update the default endpoint for the service.
If the default endpoint is specified in the resource configuration, and it is the same as in the state, then no action will be taken.
If the default endpoint is not specified in the resource configuration, and it is not in the state, then no action will be taken.

## READ
If the default endpoint is specified in the state, then the provider will read the default endpoint for the service.
If the default endpoint is not specified in the plan, then the provider will not read the default endpoint for the service.


