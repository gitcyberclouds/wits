//go:build windows

package proc

import "github.com/pranshuparmar/witr/pkg/model"

// ListAllOpenFiles is unsupported on Windows. The handle enumeration APIs
// (NtQuerySystemInformation with SystemHandleInformation) require additional
// kernel-side resolution and are out of scope here; the Locks tab is hidden
// on Windows anyway.
func ListAllOpenFiles() []*model.LockedFile { return nil }
