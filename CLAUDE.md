# CLAUDE

## Code Styleguide
- Do not abbreviate names for the variables, receiver, methods and structs. The only exception is context (ctx).
- If there are more than 3 parameters for the function input, please create a struct for the input, and copy the function/method name with the `Input` suffix. Keep the context as the first parameter and separate from the input.
- Define sentinel errors where necessary for interfaces.
- When writing new features, please write unit tests for it, particularly for abstract interfaces/implementations. Please ensure the test file package name has `_test`, and use `GoConvey` for the test framework. Use Behaviour Driven Development (BDD) methodology when writing test assertions.
- If the error is not client/validation side and is not expected at all within server side, use `fatal.OnError(err)`.
- When implementing a http api endpoint, make use of the library that handles response errors:
```go
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
```
- When handling errors in http api, please merrify them with appropriate status codes.
- If the code is seen more than once, make it a helper function
- If the code is seen more than once across packages of that service, move it to its internal package with a suitable package name
- If the package is used across different services, move it to the outer internal package with a suitable package name
- If external services are expected to communicate to a service, make sure the service defines public package with a front interface, along with its sentinel errors. Move these from private to public where possible to avoid duplicate code.
- Comments lie, so avoid writing comments where it's simpler to read, but do write comments where justified.
- Try to avoid white space where appropriate.

## Git Versioning
- Any changes should be summarised for commit, and pushed to remote.