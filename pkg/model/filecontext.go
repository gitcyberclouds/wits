package model

// FileContext holds file-related context for a process
type FileContext struct {
  // Number of open file descriptors
  OpenFiles int

  // File descriptor limit for the process
  FileLimit int

  // Files with locks held by this process
  LockedFiles []string

  // Directories being watched (if detectable)
  WatchedDirs []string
}
