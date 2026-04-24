# Boundaries

Constraints categorized by autonomy level.

## Always Do (Autonomous)

Actions the agent can take without asking.

- Run tests before committing
- Format code according to project standards
- Fix lint warnings
- Run `go vet` and `go test -race` before commits

## Ask First (Requires Confirmation)

Actions that need user approval.

- Adding new dependencies
- Changing public API signatures
- Modifying CI/CD configuration
- Database schema changes
- Deleting files or directories

## Never Do (Hard Stops)

Actions that are always prohibited.

- Commit secrets, API keys, or credentials
- Force push to main/master branch
- Skip tests (--no-verify)
- Disable security checks
- Remove error handling
