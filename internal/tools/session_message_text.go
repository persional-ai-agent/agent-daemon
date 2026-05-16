package tools

import "fmt"

func SessionSwitchedEN(prev, next string) string {
	return "_Session switched: " + prev + " -> " + next + "_"
}

func SessionResumedEN(prev, next string) string {
	return "_Session resumed: " + prev + " -> " + next + "_"
}

func ContextRecoveredEN(prev, next string) string {
	return "_Context recovered: " + prev + " -> " + next + "; replaying last input._"
}

func RetryReplayingEN() string {
	return "_Replaying latest input..._"
}

func UndoCompleteEN(removed int, next string) string {
	return fmt.Sprintf("_Undo complete: removed=%d, session switched to %s_", removed, next)
}

func ContextClearedEN(prev, next string) string {
	return "_Context cleared: " + prev + " -> " + next + "_"
}

func SessionReloadedEN(sessionID string, messageCount int) string {
	return fmt.Sprintf("_Reloaded session: %s (messages=%d)_", sessionID, messageCount)
}

func SessionSavedEN(sessionID, path string) string {
	return "_Saved session: " + sessionID + " -> " + path + "_"
}

func SessionCompressedEN(before, after int) string {
	return fmt.Sprintf("_Compressed: before=%d, after=%d, dropped=%d_", before, after, before-after)
}
