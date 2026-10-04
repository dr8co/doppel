//go:build darwin && cgo && !ios

package trash

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation

#import <Foundation/Foundation.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>

// trash_error carries failure details back to Go. `message` is allocated
// with strdup() and MUST be released by the caller with free().
typedef struct {
	int   posix_errno; // underlying POSIX errno, or 0 if unknown
	long  cocoa_code;  // NSError.code
	char *message;     // localizedDescription, or NULL
} trash_error;

// trash_item moves the item at the file-system representation `path` to the
// trash. All Objective-C objects live inside the autorelease pool (this runs
// on an arbitrary Go-managed thread with no ambient pool) and are managed by
// ARC; nothing Objective-C escapes, only plain C values and one malloc'd
// string.
static bool trash_item(const char *path, bool is_dir, trash_error *out) {
	@autoreleasepool {
		NSURL *url = [NSURL fileURLWithFileSystemRepresentation:path
		                                            isDirectory:is_dir
		                                          relativeToURL:nil];
		if (url == nil) {
			out->message = strdup("invalid path");
			return false;
		}
		NSError *err = nil;
		BOOL ok = [[NSFileManager defaultManager] trashItemAtURL:url
		                                        resultingItemURL:nil
		                                                   error:&err];
		if (ok) {
			return true;
		}
		if (err != nil) {
			out->cocoa_code = (long)err.code;
			NSError *under = err.userInfo[NSUnderlyingErrorKey];
			if ([under.domain isEqualToString:NSPOSIXErrorDomain]) {
				out->posix_errno = (int)under.code;
			} else if ([err.domain isEqualToString:NSPOSIXErrorDomain]) {
				out->posix_errno = (int)err.code;
			}
			const char *desc = err.localizedDescription.UTF8String;
			if (desc != NULL) {
				out->message = strdup(desc); // copy before the pool drains
			}
		}
		return false;
	}
}
*/
import "C"

import (
	"io/fs"
	"strconv"
	"syscall"
	"unsafe"
)

const trashSupported bool = true

func newBackend() backend { return darwinBackend{} }

type darwinBackend struct{}

func (darwinBackend) move(abs string, fi fs.FileInfo) error {
	cpath := C.CString(abs)
	defer C.free(unsafe.Pointer(cpath))

	// Zero value: no Go pointers inside, so it may be passed to C. C stores a
	// malloc'd pointer in it, which we free below.
	var cerr C.trash_error
	if C.trash_item(cpath, C.bool(fi.IsDir()), &cerr) {
		return nil
	}
	msg := C.GoString(cerr.message) // copies; "" for NULL
	if cerr.message != nil {
		C.free(unsafe.Pointer(cerr.message))
	}
	return &nsError{
		code:  int(cerr.cocoa_code),
		errno: syscall.Errno(cerr.posix_errno),
		msg:   msg,
	}
}

// nsError wraps an NSError so errors.Is works with fs.ErrNotExist,
// fs.ErrPermission, syscall errnos and ErrNotTrashable.
type nsError struct {
	code  int
	errno syscall.Errno
	msg   string
}

func (e *nsError) Error() string {
	msg := e.msg
	if msg == "" {
		msg = "NSFileManager failed to trash the item"
	}
	return msg + " (NSCocoaErrorDomain " + strconv.Itoa(e.code) + ")"
}

func (e *nsError) Unwrap() error {
	if e.errno != 0 {
		return e.errno
	}
	switch e.code {
	case 4, 260: // NSFileNoSuchFileError, NSFileReadNoSuchFileError
		return fs.ErrNotExist
	case 257, 513: // NSFileReadNoPermissionError, NSFileWriteNoPermissionError
		return fs.ErrPermission
	case 3328: // NSFeatureUnsupportedError: volume without trash support
		return ErrNotTrashable
	}
	return nil
}
