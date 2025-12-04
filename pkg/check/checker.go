package check

// Checker is implemented by all check types.
// Each check validates a specific aspect of the environment
// and returns a Result indicating success or failure.
//
// Implementations:
//   - cmdcheck.Check: verifies command existence and version
//   - envcheck.Check: validates environment variables
//   - filecheck.Check: checks file/directory properties
//   - tcpcheck.Check: tests TCP connectivity
//   - usercheck.Check: verifies user existence and properties
type Checker interface {
	Run() Result
}
