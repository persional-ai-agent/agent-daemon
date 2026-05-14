package tea

func init() {
	// Local patch: disable startup background probing.
	//
	// Reason:
	// - In our CLI runtime this probe can emit OSC/CPR queries on startup
	//   and leave the terminal input path in a bad state (can't type until
	//   extra terminal interaction like mouse wheel).
	// - We explicitly configure lipgloss/termenv in app code, so this eager
	//   init-time probe is unnecessary for our usage.
}
