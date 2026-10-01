//go:build cgo
package clean

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation
#import <Foundation/Foundation.h>

bool RecycleFile(const char* filePath) {
    @autoreleasepool {
        NSString* pathString = [NSString stringWithUTF8String:filePath];
        NSURL* fileURL = [NSURL fileURLWithPath:pathString];
        NSFileManager* fileManager = [NSFileManager defaultManager];

        NSError* error = nil;
        bool success = [fileManager trashItemAtURL:fileURL resultingItemURL:nil error:&error];

        if (!success && error) {
            NSLog(@"Error moving file to trash: %@", [error localizedDescription]);
        }
        return success;
    }
}
*/
import "C"
import (
        "fmt"
        "unsafe"
        "path/filepath"
)


func (osTrash) Move(path string) error {
        absPath, err := filepath.Abs(path)
        if err != nil {
                return fmt.Errorf("error resolving absolute path: %w", err)
        }
        cPath := C.CString(absPath)
        defer C.free(unsafe.Pointer(cPath))

        success := C.RecycleFile(cPath)
        if !success {
                return fmt.Errorf("failed to move %s to trash", path)
        }
        return nil
}
